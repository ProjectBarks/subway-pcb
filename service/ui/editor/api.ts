export async function apiFetch(path: string, opts?: RequestInit) {
	const resp = await fetch(path, {
		...opts,
		headers: { "Content-Type": "application/json", ...opts?.headers },
	});
	if (!resp.ok) throw new Error(`API ${resp.status}`);
	if (resp.status === 204) return null;
	return resp.json();
}

export const pluginsApi = {
	list: () => apiFetch("/api/v1/plugins?author=me"),
	create: (data: Record<string, unknown>) =>
		apiFetch("/api/v1/plugins", { method: "POST", body: JSON.stringify(data) }),
	update: (id: string, data: Record<string, unknown>) =>
		apiFetch(`/api/v1/plugins/${id}`, {
			method: "PUT",
			body: JSON.stringify(data),
		}),
	delete: (id: string) =>
		apiFetch(`/api/v1/plugins/${id}`, { method: "DELETE" }),
	publish: (id: string) =>
		apiFetch(`/api/v1/plugins/${id}/publish`, { method: "POST" }),
};
