export interface ConfigField {
	key: string;
	label: string;
	type: "color" | "number" | "select";
	default: string;
	group: string;
	min: string;
	max: string;
	options: string[];
}

export interface Plugin {
	id: string;
	name: string;
	lua_source: string;
	description: string;
	category: string;
	config_fields: ConfigField[];
	is_published: boolean;
	updated_at: string;
	saved: boolean;
}

export interface ConsoleMessage {
	id: string;
	type: "error" | "success" | "info" | "warning";
	message: string;
	timestamp: Date;
}
