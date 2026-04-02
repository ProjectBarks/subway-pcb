import type { FirmwareRelease } from "./types";

const REPO = "ProjectBarks/subway-pcb";
const API_BASE = "https://api.github.com";

interface GitHubAsset {
	name: string;
	browser_download_url: string;
}

interface GitHubRelease {
	draft: boolean;
	tag_name: string;
	name: string | null;
	published_at: string;
	body: string | null;
	assets: GitHubAsset[];
}

export async function fetchReleases(): Promise<FirmwareRelease[]> {
	const resp = await fetch(`${API_BASE}/repos/${REPO}/releases`, {
		headers: { Accept: "application/vnd.github.v3+json" },
	});

	if (!resp.ok) throw new Error(`GitHub API error: ${resp.status}`);

	const data: GitHubRelease[] = await resp.json();

	return data
		.filter(
			(r) =>
				!r.draft &&
				r.assets?.some((a: GitHubAsset) => a.name === "firmware.bin"),
		)
		.map((r) => ({
			tag: r.tag_name,
			name: r.name || r.tag_name,
			date: r.published_at,
			body: r.body || "",
			// biome-ignore lint/style/noNonNullAssertion: filtered to only releases with firmware.bin asset
			firmwareUrl: r.assets.find((a: GitHubAsset) => a.name === "firmware.bin")!
				.browser_download_url,
		}));
}

export async function downloadFirmware(
	release: FirmwareRelease,
): Promise<ArrayBuffer> {
	const resp = await fetch(release.firmwareUrl);
	if (!resp.ok) throw new Error(`Failed to download firmware: ${resp.status}`);
	return resp.arrayBuffer();
}
