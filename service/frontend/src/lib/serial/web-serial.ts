export class BoardSerial {
	private port: SerialPort | null = null;
	private reader: ReadableStreamDefaultReader<string> | null = null;
	private readBuffer = "";

	async connect(): Promise<boolean> {
		try {
			this.port = await navigator.serial.requestPort();
			await this.port.open({ baudRate: 115200 });
			// Set up text decoder stream for reading
			const decoder = new TextDecoderStream();
			const readable = this.port.readable!;
			(readable as ReadableStream<Uint8Array>).pipeTo(
				decoder.writable as WritableStream<Uint8Array>,
			);
			this.reader = decoder.readable.getReader();
			return true;
		} catch {
			this.port = null;
			return false;
		}
	}

	async disconnect(): Promise<void> {
		if (this.reader) {
			try {
				this.reader.cancel();
			} catch {
				// ignore
			}
			this.reader = null;
		}
		if (this.port) {
			try {
				await this.port.close();
			} catch {
				// ignore
			}
			this.port = null;
		}
	}

	async send(command: string): Promise<void> {
		if (!this.port?.writable) return;
		const encoder = new TextEncoder();
		const writer = this.port.writable.getWriter();
		try {
			await writer.write(encoder.encode(command));
		} finally {
			writer.releaseLock();
		}
	}

	async readLine(): Promise<string> {
		if (!this.reader) throw new Error("Not connected");
		while (!this.readBuffer.includes("\n")) {
			const { value, done } = await this.reader.read();
			if (done) throw new Error("Stream closed");
			this.readBuffer += value;
		}
		const idx = this.readBuffer.indexOf("\n");
		const line = this.readBuffer.slice(0, idx);
		this.readBuffer = this.readBuffer.slice(idx + 1);
		return line;
	}

	isConnected(): boolean {
		return this.port !== null;
	}
}
