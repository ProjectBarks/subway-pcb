import type { SerialResponse } from "./types";
import { BoardSerial } from "./web-serial";

const SOH_PREFIX = "\x01>";
const DEFAULT_TIMEOUT = 5000;

export class SerialProtocol extends EventTarget {
	private serial: BoardSerial;
	private seq = 0;
	private pending = new Map<
		number,
		{
			resolve: (resp: SerialResponse) => void;
			reject: (err: Error) => void;
			timer: ReturnType<typeof setTimeout>;
		}
	>();
	private running = false;

	constructor(serial?: BoardSerial) {
		super();
		this.serial = serial ?? new BoardSerial();
	}

	get connected(): boolean {
		return this.serial.isConnected();
	}

	get port(): SerialPort | null {
		return this.serial.port;
	}

	async connect(): Promise<void> {
		await this.serial.connect();
		this.running = true;
		this.readLoop();
	}

	async disconnect(): Promise<void> {
		this.running = false;
		// Reject all pending requests
		for (const [, p] of this.pending) {
			clearTimeout(p.timer);
			p.reject(new Error("Disconnected"));
		}
		this.pending.clear();
		await this.serial.disconnect();
	}

	async send<T = unknown>(
		cmd: string,
		timeout = DEFAULT_TIMEOUT,
	): Promise<SerialResponse<T>> {
		const seq = ++this.seq;
		const line = cmd.includes("#") ? cmd : `${cmd} #${seq}`;

		return new Promise((resolve, reject) => {
			const timer = setTimeout(() => {
				this.pending.delete(seq);
				reject(new Error(`Timeout waiting for response to: ${cmd}`));
			}, timeout);

			this.pending.set(seq, {
				resolve: resolve as (r: SerialResponse) => void,
				reject,
				timer,
			});
			this.serial.send(`${line}\n`).catch(reject);
		});
	}

	async sendRaw(line: string): Promise<void> {
		await this.serial.send(`${line}\n`);
	}

	private async readLoop(): Promise<void> {
		while (this.running && this.serial.isConnected()) {
			try {
				const line = await this.serial.readLine();
				if (!line) continue;

				const sohIdx = line.indexOf(SOH_PREFIX);
				if (sohIdx >= 0) {
					// Protocol response — SOH may not be at position 0 due to
					// ESP_LOG output interleaving on the shared UART
					const json = line.slice(sohIdx + SOH_PREFIX.length);
					try {
						const resp: SerialResponse = JSON.parse(json);
						const pending = this.pending.get(resp.seq);
						if (pending) {
							clearTimeout(pending.timer);
							this.pending.delete(resp.seq);
							pending.resolve(resp);
						}
					} catch {
						// Malformed JSON, emit as log
						this.dispatchEvent(new CustomEvent("log", { detail: line }));
					}
				} else {
					// ESP_LOG line
					this.dispatchEvent(new CustomEvent("log", { detail: line }));
				}
			} catch {
				if (this.running) {
					this.dispatchEvent(new CustomEvent("disconnect"));
					break;
				}
			}
		}
	}
}
