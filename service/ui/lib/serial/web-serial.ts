export class BoardSerial {
	private _port: SerialPort | null = null;
	private reader: ReadableStreamDefaultReader<string> | null = null;
	private readBuffer = "";
	private running = false;

	onDisconnect: (() => void) | null = null;

	get port(): SerialPort | null {
		return this._port;
	}

	async connect(): Promise<boolean> {
		try {
			this._port = await navigator.serial.requestPort();
			await this._port.open({ baudRate: 115200 });
			this.running = true;

			// Set up text decoder stream for reading
			const decoder = new TextDecoderStream();
			// biome-ignore lint/style/noNonNullAssertion: readable is available after successful open()
			const readable = this._port.readable!;
			(readable as ReadableStream<Uint8Array>).pipeTo(
				decoder.writable as WritableStream<Uint8Array>,
			);
			this.reader = decoder.readable.getReader();

			// Listen for disconnect
			navigator.serial.addEventListener("disconnect", (e) => {
				if ((e as Event & { target: SerialPort }).target === this._port) {
					this.running = false;
					this._port = null;
					this.reader = null;
					this.onDisconnect?.();
				}
			});

			return true;
		} catch {
			this._port = null;
			return false;
		}
	}

	async disconnect(): Promise<void> {
		this.running = false;
		if (this.reader) {
			try {
				await this.reader.cancel();
			} catch {
				// ignore
			}
			this.reader = null;
		}
		if (this._port) {
			try {
				await this._port.close();
			} catch {
				// ignore
			}
			this._port = null;
		}
	}

	async send(command: string): Promise<void> {
		if (!this._port?.writable) return;
		const encoder = new TextEncoder();
		const writer = this._port.writable.getWriter();
		try {
			await writer.write(encoder.encode(command));
		} finally {
			writer.releaseLock();
		}
	}

	async readLine(): Promise<string> {
		if (!this.reader) throw new Error("Not connected");
		while (!this.readBuffer.includes("\n")) {
			try {
				const { value, done } = await this.reader.read();
				if (done) throw new Error("Stream closed");
				this.readBuffer += value;
			} catch (e) {
				if (!this.running) {
					// Reader was cancelled during disconnect, propagate gracefully
					throw new Error("Stream closed");
				}
				throw e;
			}
		}
		const idx = this.readBuffer.indexOf("\n");
		const line = this.readBuffer.slice(0, idx);
		this.readBuffer = this.readBuffer.slice(idx + 1);
		return line;
	}

	isConnected(): boolean {
		return this._port !== null;
	}
}
