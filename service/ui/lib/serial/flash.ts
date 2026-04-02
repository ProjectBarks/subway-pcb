import type { FlashProgress } from "./types";

// esptool-js is dynamically imported to avoid bundling it when not needed
export async function flashFirmware(
	port: SerialPort,
	firmwareBin: ArrayBuffer,
	onProgress: (p: FlashProgress) => void,
): Promise<void> {
	const { ESPLoader, Transport } = await import("esptool-js");

	const transport = new Transport(port, true);

	onProgress({
		phase: "connecting",
		percent: 0,
		message: "Entering download mode...",
	});

	const loader = new ESPLoader({
		transport,
		baudrate: 460800,
		romBaudrate: 115200,
	});

	await loader.main();

	onProgress({
		phase: "erasing",
		percent: 0,
		message: "Erasing flash...",
	});

	// Convert ArrayBuffer to binary string for esptool-js
	const uint8 = new Uint8Array(firmwareBin);
	let binaryString = "";
	for (let i = 0; i < uint8.length; i++) {
		binaryString += String.fromCharCode(uint8[i]);
	}

	onProgress({
		phase: "writing",
		percent: 0,
		message: "Flashing...",
	});

	await loader.writeFlash({
		fileArray: [{ data: binaryString, address: 0x10000 }],
		flashSize: "keep",
		compress: true,
		reportProgress: (_idx: number, written: number, total: number) => {
			const pct = Math.round((written / total) * 100);
			onProgress({
				phase: "writing",
				percent: pct,
				message: `Flashing... ${pct}%`,
			});
		},
	});

	onProgress({
		phase: "verifying",
		percent: 100,
		message: "Verifying...",
	});

	onProgress({
		phase: "done",
		percent: 100,
		message: "Flash complete. Rebooting...",
	});
	await loader.hardReset();
}
