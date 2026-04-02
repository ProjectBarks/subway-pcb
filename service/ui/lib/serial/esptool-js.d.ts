declare module "esptool-js" {
	export class Transport {
		constructor(port: SerialPort, tracing?: boolean);
	}

	export interface ESPLoaderOptions {
		transport: Transport;
		baudrate: number;
		romBaudrate?: number;
	}

	export interface FlashFile {
		data: string;
		address: number;
	}

	export interface WriteFlashOptions {
		fileArray: FlashFile[];
		flashSize: string;
		compress?: boolean;
		reportProgress?: (
			fileIndex: number,
			written: number,
			total: number,
		) => void;
	}

	export class ESPLoader {
		constructor(options: ESPLoaderOptions);
		main(): Promise<void>;
		writeFlash(options: WriteFlashOptions): Promise<void>;
		hardReset(): Promise<void>;
	}
}
