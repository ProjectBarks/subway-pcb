import { render } from "preact";
import { useEffect, useRef, useState } from "preact/hooks";
import { type BoardViewerHandle, initBoardViewer } from "../lib/board-viewer";
import { pluginsApi } from "./api";
import { CodeEditor } from "./code-editor";
import { ConfigFieldEditor } from "./config-editor";
import type { ConfigField, ConsoleMessage, Plugin } from "./types";

const DEFAULT_LUA = `-- LED Plugin: Rainbow Wave
function render(t, leds)
  for i = 0, leds - 1 do
    local hue = (i / leds + t * 0.1) % 1.0
    local r, g, b = hsv_to_rgb(hue, 1.0, 1.0)
    set_led(i, r, g, b)
  end
end

function hsv_to_rgb(h, s, v)
  local i = math.floor(h * 6)
  local f = h * 6 - i
  local p = v * (1 - s)
  local q = v * (1 - f * s)
  local t = v * (1 - (1 - f) * s)
  i = i % 6
  if i == 0 then return v, t, p
  elseif i == 1 then return q, v, p
  elseif i == 2 then return p, v, t
  elseif i == 3 then return p, q, v
  elseif i == 4 then return t, p, v
  else return v, p, q end
end`;

function EditorPreview({ isRunning }: { isRunning: boolean }) {
	const containerRef = useRef<HTMLDivElement>(null);
	const viewerRef = useRef<BoardViewerHandle | null>(null);

	useEffect(() => {
		if (!containerRef.current) return;
		initBoardViewer(containerRef.current, {
			boardUrl: "/static/dist/boards/nyc-subway/v1/board.json",
			mode: "preview",
		}).then((handle) => {
			viewerRef.current = handle;
		});
		return () => {
			viewerRef.current?.dispose();
			viewerRef.current = null;
		};
	}, []);

	return (
		<div class="hidden lg:flex flex-col bg-bg-primary">
			<div class="px-5 py-3 border-b border-border-subtle bg-bg-surface flex items-center justify-between">
				<div class="flex items-center gap-2">
					<svg
						class="size-4 text-text-secondary"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
					>
						<path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0" />
						<circle cx="12" cy="12" r="3" />
					</svg>
					<span class="text-text-primary text-sm font-medium">
						Live Preview
					</span>
				</div>
				{isRunning && (
					<div class="flex items-center gap-2 text-xs text-text-muted">
						<div class="size-2 rounded-full bg-status-online animate-pulse" />
						<span>Rendering</span>
					</div>
				)}
			</div>
			<div ref={containerRef} class="flex-1" />
		</div>
	);
}

function EditorApp() {
	const [plugins, setPlugins] = useState<Plugin[]>([]);
	const [selectedPluginId, setSelectedPluginId] = useState<string | null>(null);
	const [isRunning, setIsRunning] = useState(false);
	const [activeTab, setActiveTab] = useState<"code" | "config" | "info">(
		"code",
	);
	const [consoleMessages, setConsoleMessages] = useState<ConsoleMessage[]>([]);
	const [saveStatus, setSaveStatus] = useState<
		"saved" | "unsaved" | "saving" | "error"
	>("saved");
	const [sidebarOpen, setSidebarOpen] = useState(false);
	const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
	const [editingName, setEditingName] = useState<string | null>(null);
	const [editingNameValue, setEditingNameValue] = useState("");

	const selectedPlugin = plugins.find((p) => p.id === selectedPluginId);

	// Load plugins on mount
	useEffect(() => {
		pluginsApi
			.list()
			.then((data) => {
				const list: Plugin[] = (data.data || []).map(
					(p: Record<string, unknown>) => ({
						...p,
						config_fields: (p.config_fields as ConfigField[] | null) || [],
						saved: true,
					}),
				);
				setPlugins(list);
				if (list.length > 0) setSelectedPluginId(list[0].id);
			})
			.catch(() => {
				addConsoleMessage("error", "Failed to load plugins");
			});
	}, []);

	const addConsoleMessage = (type: ConsoleMessage["type"], message: string) => {
		setConsoleMessages((prev) =>
			[
				...prev,
				{
					id: crypto.randomUUID(),
					type,
					message,
					timestamp: new Date(),
				},
			].slice(-50),
		);
	};

	const createNewPlugin = async () => {
		try {
			const data = await pluginsApi.create({
				name: "Untitled Plugin",
				lua_source: DEFAULT_LUA,
				category: "ambient",
			});
			const newPlugin: Plugin = {
				...data.data,
				config_fields: data.data.config_fields || [],
				saved: true,
			};
			setPlugins((prev) => [...prev, newPlugin]);
			setSelectedPluginId(newPlugin.id);
			setSaveStatus("saved");
			addConsoleMessage("info", `Created new plugin: ${newPlugin.name}`);
		} catch {
			addConsoleMessage("error", "Failed to create plugin");
		}
	};

	const updatePlugin = (updates: Partial<Plugin>) => {
		if (!selectedPluginId) return;
		setPlugins((prev) =>
			prev.map((p) =>
				p.id === selectedPluginId ? { ...p, ...updates, saved: false } : p,
			),
		);
		setSaveStatus("unsaved");
	};

	const deletePlugin = async (id: string) => {
		try {
			await pluginsApi.delete(id);
			setPlugins((prev) => prev.filter((p) => p.id !== id));
			if (selectedPluginId === id) {
				const remaining = plugins.filter((p) => p.id !== id);
				setSelectedPluginId(remaining.length > 0 ? remaining[0].id : null);
			}
			addConsoleMessage("info", "Plugin deleted");
		} catch {
			addConsoleMessage("error", "Failed to delete plugin");
		}
		setDeleteTarget(null);
	};

	const savePlugin = async () => {
		if (!selectedPlugin) return;
		setSaveStatus("saving");
		if (!selectedPlugin.lua_source.includes("function render")) {
			setSaveStatus("error");
			addConsoleMessage("error", "Missing required 'render' function");
			return;
		}
		try {
			const data = await pluginsApi.update(selectedPlugin.id, {
				name: selectedPlugin.name,
				lua_source: selectedPlugin.lua_source,
				description: selectedPlugin.description,
				category: selectedPlugin.category,
				config_fields: selectedPlugin.config_fields,
			});
			setPlugins((prev) =>
				prev.map((p) =>
					p.id === selectedPluginId ? { ...p, ...data.data, saved: true } : p,
				),
			);
			setSaveStatus("saved");
			addConsoleMessage("success", `Saved "${selectedPlugin.name}"`);
		} catch {
			setSaveStatus("error");
			addConsoleMessage("error", "Failed to save plugin");
		}
	};

	const publishPlugin = async () => {
		if (!selectedPlugin) return;
		try {
			const data = await pluginsApi.publish(selectedPlugin.id);
			setPlugins((prev) =>
				prev.map((p) =>
					p.id === selectedPlugin.id
						? { ...p, is_published: data.data.is_published }
						: p,
				),
			);
			addConsoleMessage(
				"success",
				data.data.is_published ? "Plugin published!" : "Plugin unpublished",
			);
		} catch {
			addConsoleMessage("error", "Failed to toggle publish");
		}
	};

	const handleRun = () => {
		if (!selectedPlugin) return;
		if (!selectedPlugin.lua_source.includes("function render")) {
			addConsoleMessage("error", "Missing required 'render' function");
			return;
		}
		setIsRunning(true);
		addConsoleMessage("success", "Running plugin...");
		addConsoleMessage("info", "Rendering 478 LEDs at 60fps");
	};

	return (
		<div class="flex w-full">
			{/* Mobile sidebar overlay */}
			{sidebarOpen && (
				<div
					class="lg:hidden fixed inset-0 bg-black/60 z-40 mt-16"
					onClick={() => setSidebarOpen(false)}
				/>
			)}

			{/* Sidebar */}
			<div
				class={`w-72 border-r border-border-subtle bg-bg-surface flex flex-col h-full fixed lg:static inset-y-0 left-0 mt-16 lg:mt-0 z-40 lg:z-auto transition-transform duration-300 ${sidebarOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"}`}
			>
				<div class="p-5 border-b border-border-subtle">
					<h2 class="text-text-primary font-semibold mb-3">Your Plugins</h2>
					<button
						type="button"
						onClick={createNewPlugin}
						class="w-full h-10 bg-accent-gold hover:bg-accent-gold-hover text-black rounded-lg font-medium transition-colors flex items-center justify-center gap-2"
					>
						<svg
							class="size-4"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
						>
							<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
							<polyline points="14 2 14 8 20 8" />
							<path d="M10 12l-2 2 2 2" />
							<path d="M14 12l2 2-2 2" />
						</svg>
						New Plugin
					</button>
				</div>
				<div class="flex-1 overflow-auto p-3">
					{plugins.length > 0 ? (
						<div class="space-y-1.5">
							{plugins.map((plugin) => (
								<div
									key={plugin.id}
									class={`group relative rounded-lg transition-all cursor-pointer ${
										selectedPluginId === plugin.id
											? "bg-accent-gold/10 border border-accent-gold/30"
											: "hover:bg-bg-surface-raised border border-transparent"
									}`}
								>
									{editingName === plugin.id ? (
										<div class="p-3">
											<input
												type="text"
												value={editingNameValue}
												onInput={(e) =>
													setEditingNameValue(
														(e.target as HTMLInputElement).value,
													)
												}
												onBlur={() => {
													if (editingNameValue.trim()) {
														setPlugins((prev) =>
															prev.map((p) =>
																p.id === plugin.id
																	? {
																			...p,
																			name: editingNameValue.trim(),
																			saved: false,
																		}
																	: p,
															),
														);
														setSaveStatus("unsaved");
													}
													setEditingName(null);
												}}
												onKeyDown={(e) => {
													if (e.key === "Enter")
														(e.target as HTMLInputElement).blur();
													if (e.key === "Escape") setEditingName(null);
												}}
												class="w-full px-3 py-1.5 bg-bg-primary border border-accent-gold rounded-lg text-text-primary text-sm focus:outline-none focus:ring-1 focus:ring-accent-gold"
												autoFocus
											/>
										</div>
									) : (
										<button
											type="button"
											onClick={() => {
												setSelectedPluginId(plugin.id);
												setSidebarOpen(false);
											}}
											class="w-full text-left p-3 pr-10"
										>
											<div class="flex items-center gap-2 mb-1">
												<span class="text-text-primary text-sm font-medium truncate">
													{plugin.name}
												</span>
												{!plugin.saved && (
													<div class="size-1.5 rounded-full bg-accent-gold flex-shrink-0" />
												)}
											</div>
											<div class="text-text-muted text-xs flex items-center gap-2">
												{plugin.is_published && (
													<span class="text-status-online">Published</span>
												)}
											</div>
										</button>
									)}
									{/* Context menu button */}
									<div class="absolute top-2.5 right-2 opacity-0 group-hover:opacity-100 transition-opacity flex gap-1">
										<button
											type="button"
											onClick={(e) => {
												e.stopPropagation();
												setEditingName(plugin.id);
												setEditingNameValue(plugin.name);
											}}
											class="size-7 rounded-md hover:bg-bg-surface-raised flex items-center justify-center"
											title="Rename"
										>
											<svg
												class="size-3.5 text-text-secondary"
												viewBox="0 0 24 24"
												fill="none"
												stroke="currentColor"
												stroke-width="2"
											>
												<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
											</svg>
										</button>
										<button
											type="button"
											onClick={(e) => {
												e.stopPropagation();
												setDeleteTarget(plugin.id);
											}}
											class="size-7 rounded-md hover:bg-status-error/10 flex items-center justify-center"
											title="Delete"
										>
											<svg
												class="size-3.5 text-status-error"
												viewBox="0 0 24 24"
												fill="none"
												stroke="currentColor"
												stroke-width="2"
											>
												<path d="M3 6h18" />
												<path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" />
												<path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
											</svg>
										</button>
									</div>
								</div>
							))}
						</div>
					) : (
						<div class="flex flex-col items-center justify-center h-full text-center px-6">
							<div class="size-16 rounded-2xl bg-bg-surface-raised border border-border-subtle flex items-center justify-center mb-4">
								<svg
									class="size-8 text-text-muted"
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="2"
								>
									<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
									<polyline points="14 2 14 8 20 8" />
									<path d="M10 12l-2 2 2 2" />
									<path d="M14 12l2 2-2 2" />
								</svg>
							</div>
							<p class="text-text-secondary text-sm mb-1">No plugins yet</p>
							<p class="text-text-muted text-xs">Click "New Plugin" to start</p>
						</div>
					)}
				</div>
			</div>

			{/* Main editor area */}
			<div class="flex-1 flex flex-col">
				{selectedPlugin ? (
					<>
						{/* Toolbar */}
						<div class="border-b border-border-subtle bg-bg-surface/80 backdrop-blur-xl px-5 lg:px-10">
							<div class="h-16 flex items-center justify-between">
								<div class="flex items-center gap-2 md:gap-3 flex-1 overflow-hidden">
									<button
										type="button"
										onClick={() => setSidebarOpen(!sidebarOpen)}
										class="lg:hidden size-9 rounded-md bg-bg-surface border border-border-subtle flex items-center justify-center"
									>
										<svg
											class="size-4 text-text-secondary"
											viewBox="0 0 24 24"
											fill="none"
											stroke="currentColor"
											stroke-width="2"
										>
											<line x1="4" x2="20" y1="12" y2="12" />
											<line x1="4" x2="20" y1="6" y2="6" />
											<line x1="4" x2="20" y1="18" y2="18" />
										</svg>
									</button>
									<input
										type="text"
										value={selectedPlugin.name}
										onInput={(e) =>
											updatePlugin({
												name: (e.target as HTMLInputElement).value,
											})
										}
										class="text-base md:text-xl font-semibold text-text-primary bg-transparent border-none focus:outline-none focus:ring-1 focus:ring-accent-gold rounded px-2 py-1 max-w-[200px] md:max-w-none"
									/>
									<div class="hidden md:flex items-center gap-2 text-xs text-text-muted">
										{saveStatus === "saved" && (
											<>
												<svg
													class="size-3.5 text-status-online"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2"
												>
													<path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
													<path d="m9 11 3 3L22 4" />
												</svg>
												<span>Saved</span>
											</>
										)}
										{saveStatus === "unsaved" && (
											<>
												<svg
													class="size-3.5 text-accent-gold"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2"
												>
													<circle cx="12" cy="12" r="10" />
													<polyline points="12 6 12 12 16 14" />
												</svg>
												<span>Unsaved</span>
											</>
										)}
										{saveStatus === "saving" && (
											<>
												<svg
													class="size-3.5 text-accent-gold animate-spin"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2"
												>
													<circle cx="12" cy="12" r="10" />
													<polyline points="12 6 12 12 16 14" />
												</svg>
												<span>Saving...</span>
											</>
										)}
										{saveStatus === "error" && (
											<>
												<svg
													class="size-3.5 text-status-error"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2"
												>
													<circle cx="12" cy="12" r="10" />
													<line x1="12" x2="12" y1="8" y2="12" />
													<line x1="12" x2="12.01" y1="16" y2="16" />
												</svg>
												<span>Error</span>
											</>
										)}
									</div>
								</div>
								<div class="flex items-center gap-2 md:gap-3">
									<button
										type="button"
										onClick={handleRun}
										class="h-10 px-4 md:px-6 bg-accent-gold text-black rounded-full flex items-center gap-2 hover:scale-[0.98] transition-transform text-sm"
									>
										<svg
											class="size-4"
											viewBox="0 0 24 24"
											fill="currentColor"
											stroke="none"
										>
											<polygon points="6 3 20 12 6 21 6 3" />
										</svg>
										<span class="hidden sm:inline">Run</span>
									</button>
									<button
										type="button"
										onClick={savePlugin}
										disabled={saveStatus === "saving"}
										class="h-10 px-4 md:px-6 bg-bg-surface border border-border-subtle rounded-full text-text-primary hover:border-border-hover transition-colors flex items-center gap-2 disabled:opacity-50 text-sm"
									>
										<svg
											class="size-4"
											viewBox="0 0 24 24"
											fill="none"
											stroke="currentColor"
											stroke-width="2"
										>
											<path d="M15.2 3a2 2 0 0 1 1.4.6l3.8 3.8a2 2 0 0 1 .6 1.4V19a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z" />
											<path d="M17 21v-7a1 1 0 0 0-1-1H8a1 1 0 0 0-1 1v7" />
											<path d="M7 3v4a1 1 0 0 0 1 1h7" />
										</svg>
										<span class="hidden sm:inline">Save</span>
									</button>
									<button
										type="button"
										onClick={publishPlugin}
										disabled={!selectedPlugin.saved}
										class="hidden md:flex h-10 px-6 bg-white text-black rounded-full hover:scale-[0.98] transition-transform items-center gap-2 disabled:opacity-50 text-sm"
									>
										<svg
											class="size-4"
											viewBox="0 0 24 24"
											fill="none"
											stroke="currentColor"
											stroke-width="2"
										>
											<path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8" />
											<polyline points="16 6 12 2 8 6" />
											<line x1="12" x2="12" y1="2" y2="15" />
										</svg>
										{selectedPlugin.is_published ? "Unpublish" : "Publish"}
									</button>
								</div>
							</div>
						</div>

						{/* Editor + Preview layout */}
						<div class="lg:grid lg:grid-cols-2 flex-1 flex flex-col overflow-hidden">
							{/* Code editor side */}
							<div class="lg:border-r border-border-subtle flex flex-col flex-1 overflow-hidden">
								{/* Tabs */}
								<div class="flex gap-1 px-5 py-3 border-b border-border-subtle bg-bg-surface">
									{(["code", "config", "info"] as const).map((tab) => (
										<button
											type="button"
											key={tab}
											onClick={() => setActiveTab(tab)}
											class={`h-9 px-4 rounded-md text-sm font-medium transition-colors ${
												activeTab === tab
													? "bg-bg-surface-raised text-text-primary"
													: "text-text-secondary hover:text-text-primary"
											}`}
										>
											{tab === "code"
												? "Code"
												: tab === "config"
													? "Config"
													: "Info"}
										</button>
									))}
								</div>

								{/* Tab content */}
								<div class="flex-1 overflow-auto">
									{activeTab === "code" && (
										<CodeEditor
											value={selectedPlugin.lua_source}
											onChange={(lua_source) => updatePlugin({ lua_source })}
										/>
									)}
									{activeTab === "config" && selectedPlugin && (
										<ConfigFieldEditor
											fields={selectedPlugin.config_fields}
											onChange={(config_fields) =>
												updatePlugin({ config_fields } as Partial<Plugin>)
											}
										/>
									)}
									{activeTab === "info" && (
										<div class="p-5 bg-bg-primary space-y-4">
											<h3 class="text-text-primary mb-4">Plugin Information</h3>
											<div>
												<label class="text-text-muted block mb-2">
													Description
												</label>
												<textarea
													placeholder="Describe what this plugin does..."
													rows={4}
													value={selectedPlugin.description}
													onInput={(e) =>
														updatePlugin({
															description: (e.target as HTMLTextAreaElement)
																.value,
														})
													}
													class="w-full p-4 bg-bg-surface border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted resize-none focus:outline-none focus:ring-1 focus:ring-accent-gold"
												/>
											</div>
											<div>
												<label class="text-text-muted block mb-2">
													Category
												</label>
												<select
													value={selectedPlugin.category}
													onChange={(e) =>
														updatePlugin({
															category: (e.target as HTMLSelectElement).value,
														})
													}
													class="w-full h-10 px-4 bg-bg-surface border border-border-subtle rounded-lg text-text-primary focus:outline-none focus:ring-1 focus:ring-accent-gold"
												>
													<option value="ambient">Ambient</option>
													<option value="data-driven">Data-Driven</option>
													<option value="reactive">Reactive</option>
													<option value="artistic">Artistic</option>
												</select>
											</div>
										</div>
									)}
								</div>

								{/* Console */}
								<div class="h-32 border-t border-border-subtle bg-black flex flex-col">
									<div class="flex items-center justify-between px-4 py-2 border-b border-border-subtle">
										<span class="text-text-secondary text-xs font-medium">
											Console
										</span>
										<button
											type="button"
											onClick={() => setConsoleMessages([])}
											class="text-text-muted hover:text-text-primary text-xs"
										>
											Clear
										</button>
									</div>
									<div class="flex-1 overflow-auto p-2">
										<div class="text-xs font-mono space-y-0.5">
											{consoleMessages.map((msg) => (
												<div key={msg.id} class="flex items-start gap-2">
													<span class="text-text-muted opacity-50">
														{msg.timestamp.toLocaleTimeString()}
													</span>
													<span
														class={
															msg.type === "error"
																? "text-status-error"
																: msg.type === "success"
																	? "text-status-online"
																	: msg.type === "warning"
																		? "text-accent-gold"
																		: "text-text-secondary"
														}
													>
														{msg.message}
													</span>
												</div>
											))}
											{consoleMessages.length === 0 && (
												<div class="text-text-muted opacity-50">
													Console output will appear here...
												</div>
											)}
										</div>
									</div>
								</div>
							</div>

							{/* Preview side (desktop) */}
							<EditorPreview isRunning={isRunning} />
						</div>
					</>
				) : (
					/* No plugin selected */
					<div class="flex-1 flex items-center justify-center p-6">
						<div class="text-center">
							<div class="size-20 rounded-2xl bg-bg-surface border border-border-subtle flex items-center justify-center mx-auto mb-6">
								<svg
									class="size-10 text-text-muted"
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="2"
								>
									<path d="m18 16 4-4-4-4" />
									<path d="m6 8-4 4 4 4" />
									<path d="m14.5 4-5 16" />
								</svg>
							</div>
							<h3 class="text-text-primary mb-2">Plugin Editor</h3>
							<p class="text-text-secondary mb-6">
								Create custom LED patterns with Lua scripts
							</p>
							<button
								type="button"
								onClick={createNewPlugin}
								class="h-10 px-6 bg-accent-gold text-black rounded-full font-medium transition-all flex items-center gap-2 mx-auto"
							>
								<svg
									class="size-4"
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="2"
								>
									<path d="M5 12h14" />
									<path d="M12 5v14" />
								</svg>
								Create Your First Plugin
							</button>
						</div>
					</div>
				)}
			</div>

			{/* Delete confirmation dialog */}
			{deleteTarget && (
				<div class="fixed inset-0 bg-black/80 z-50 flex items-center justify-center">
					<div class="w-full max-w-md p-6 bg-bg-surface border border-border-subtle rounded-xl">
						<h3 class="text-text-primary text-lg font-semibold mb-2">
							Delete Plugin?
						</h3>
						<p class="text-text-secondary text-sm mb-6">
							This action cannot be undone. This will permanently delete "
							{plugins.find((p) => p.id === deleteTarget)?.name}".
						</p>
						<div class="flex gap-3 justify-end">
							<button
								type="button"
								onClick={() => setDeleteTarget(null)}
								class="h-10 px-6 bg-bg-surface-raised border border-border-subtle rounded-full text-text-primary hover:border-border-hover transition-colors"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={() => deletePlugin(deleteTarget)}
								class="h-10 px-6 bg-status-error text-white rounded-full hover:scale-[0.98] transition-transform"
							>
								Delete
							</button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}

export function mount() {
	const root = document.getElementById("editor-root");
	if (root) {
		render(<EditorApp />, root);
	}
}
