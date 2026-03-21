import type { LuaRunner, StationState } from "./lua-runner";

export interface MtaPollCallbacks {
	/** Called after a successful fetch with the station list */
	onSuccess?: (stations: StationState[]) => void;
	/** Called when the fetch fails */
	onError?: () => void;
}

/**
 * Poll the MTA state endpoint and push station data into the runner.
 * Returns a cleanup function that stops the polling interval.
 */
export function pollMtaState(
	runner: LuaRunner,
	intervalMs = 5000,
	callbacks?: MtaPollCallbacks,
): () => void {
	const fetchState = async () => {
		try {
			const resp = await fetch("/api/v1/state?format=json");
			if (resp.ok) {
				const data = await resp.json();
				if (data.stations) {
					runner.setMtaState(data.stations);
					callbacks?.onSuccess?.(data.stations);
				}
			}
		} catch {
			callbacks?.onError?.();
		}
	};
	fetchState();
	const interval = setInterval(fetchState, intervalMs);
	return () => clearInterval(interval);
}
