use anyhow::{anyhow, Result};
use chrono::{Datelike, NaiveDate, TimeZone, Utc};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::collections::HashMap;
use tokio::io::{stdin, stdout, AsyncBufReadExt, AsyncWriteExt, BufReader};
use uuid::Uuid;

use crate::api::LaputaError;
use crate::config::MempalaceConfig;
use crate::dialect::Dialect;
use crate::diary::{self, Diary, DiaryWriteRequest};
use crate::identity::IdentityInitializer;
use crate::knowledge_graph::KnowledgeGraph;
use crate::palace_graph::PalaceGraph;
use crate::searcher::{RecallQuery, Searcher, SemanticSearchOptions};
use crate::vector_storage::{UserIntervention, VectorStorage};

const MIN_RECALL_LIMIT: usize = 1;
const MAX_RECALL_LIMIT: usize = 10_000;
const MIN_ALLOWED_DATE_YEAR: i32 = 1900;
const MAX_ALLOWED_DATE_YEAR: i32 = 2100;
const MAX_TIME_RANGE_DAYS: i64 = 365;

#[derive(Debug, Serialize, Deserialize)]
struct JsonRpcRequest {
    jsonrpc: String,
    method: String,
    params: Option<Value>,
    id: Option<Value>,
}

#[derive(Debug, Serialize, Deserialize)]
struct JsonRpcResponse {
    jsonrpc: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    result: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<JsonRpcError>,
    id: Option<Value>,
}

#[derive(Debug, Serialize, Deserialize)]
struct JsonRpcError {
    code: i32,
    message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    data: Option<Value>,
}

pub struct McpServer {
    config: MempalaceConfig,
    searcher: Searcher,
    kg: KnowledgeGraph,
    pg: PalaceGraph,
    dialect: Dialect,
}

impl McpServer {
    pub async fn new(config: MempalaceConfig) -> Result<Self> {
        // Ensure config directory exists
        let _ = std::fs::create_dir_all(&config.config_dir);

        let searcher = Searcher::new(config.clone());
        let kg = open_knowledge_graph(&config.config_dir.join("knowledge.db"))?;
        let pg = PalaceGraph::new();
        // Phase 4: load external emotion map and inject into dialect
        let custom_emotions = config.load_emotions_map();
        let dialect = Dialect::with_custom_emotions(None, None, custom_emotions);

        Ok(Self {
            config,
            searcher,
            kg,
            pg,
            dialect,
        })
    }

    #[cfg(test)]
    pub(crate) fn new_test(config: MempalaceConfig) -> Self {
        let _ = std::fs::create_dir_all(&config.config_dir);
        let searcher = Searcher::new(config.clone());
        let kg_path = config.config_dir.join("test_knowledge.db");
        let kg = open_knowledge_graph(&kg_path).unwrap();
        let pg = PalaceGraph::new();
        let dialect = Dialect::default();

        Self {
            config,
            searcher,
            kg,
            pg,
            dialect,
        }
    }

    pub async fn run(&mut self) -> Result<()> {
        let mut reader = BufReader::new(stdin());
        let mut line = String::new();

        while reader.read_line(&mut line).await? > 0 {
            let req: JsonRpcRequest = match serde_json::from_str(&line) {
                Ok(r) => r,
                Err(_) => {
                    line.clear();
                    continue;
                }
            };

            // JSON-RPC notifications have no id — must NOT send a response
            let is_notification = req.id.is_none() || req.method.starts_with("notifications/");
            if is_notification {
                line.clear();
                continue;
            }

            let resp = self.handle_request(req).await;
            let resp_json = serde_json::to_string(&resp)? + "\n";
            stdout().write_all(resp_json.as_bytes()).await?;
            stdout().flush().await?;
            line.clear();
        }

        Ok(())
    }

    async fn handle_request(&mut self, req: JsonRpcRequest) -> JsonRpcResponse {
        let result = match req.method.as_str() {
            "initialize" => self.handle_initialize(req.params),
            "tools/list" => self.handle_tools_list(),
            "tools/call" => self.handle_tools_call(req.params).await,
            "resources/list" => Ok(json!({ "resources": [] })),
            "resources/read" => Err(anyhow!("Resource not found")),
            "prompts/list" => Ok(json!({ "prompts": [] })),
            // Silently return empty object for unknown but non-notification methods
            _ => Ok(json!({})),
        };

        match result {
            Ok(res) => JsonRpcResponse {
                jsonrpc: "2.0".to_string(),
                result: Some(res),
                error: None,
                id: req.id,
            },
            Err(e) => {
                let (code, message) = map_jsonrpc_error(&e);
                JsonRpcResponse {
                    jsonrpc: "2.0".to_string(),
                    result: None,
                    error: Some(JsonRpcError {
                        code,
                        message,
                        data: None,
                    }),
                    id: req.id,
                }
            }
        }
    }

    fn handle_initialize(&self, _params: Option<Value>) -> Result<Value> {
        Ok(json!({
            "protocolVersion": "2024-11-05",
            "capabilities": {
                "tools": {
                    "listChanged": true
                }
            },
            "serverInfo": {
                "name": "laputa",
                "version": env!("CARGO_PKG_VERSION")
            }
        }))
    }

    fn handle_tools_list(&self) -> Result<Value> {
        Ok(json!({
            "tools": [
                {
                    "name": "laputa_init",
                    "description": "Initialize Laputa identity and storage in the configured directory.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "user_name": { "type": "string" }
                        },
                        "required": ["user_name"]
                    }
                },
                {
                    "name": "laputa_diary_write",
                    "description": "Write a diary memory into Laputa storage.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "agent": { "type": "string" },
                            "content": { "type": "string" },
                            "tags": {
                                "type": "array",
                                "items": { "type": "string" }
                            },
                            "emotion": { "type": "string" },
                            "timestamp": { "type": "string" },
                            "wing": { "type": "string" },
                            "room": { "type": "string" }
                        },
                        "required": ["agent", "content"]
                    }
                },
                {
                    "name": "laputa_recall",
                    "description": "Recall memories by time range with optional wing and room filters.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "time_range": { "type": "string" },
                            "wing": { "type": "string" },
                            "room": { "type": "string" },
                            "limit": { "type": "integer", "default": 100 },
                            "include_discarded": { "type": "boolean", "default": false }
                        },
                        "required": ["time_range"]
                    }
                },
                {
                    "name": "laputa_wakeup_generate",
                    "description": "Generate the current wakeup pack from existing memories.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "wing": { "type": "string" }
                        }
                    }
                },
                {
                    "name": "laputa_mark_important",
                    "description": "Mark a numeric memory_id as important using the existing heat intervention flow.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "memory_id": { "type": "integer" },
                            "reason": { "type": "string", "default": "marked important via MCP" }
                        },
                        "required": ["memory_id"]
                    }
                },
                {
                    "name": "laputa_get_heat_status",
                    "description": "Return heat status fields for a numeric memory_id.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "memory_id": { "type": "integer" }
                        },
                        "required": ["memory_id"]
                    }
                },
                {
                    "name": "mempalace_status",
                    "description": "Returns total drawers, wings, rooms, protocol, and AAAK spec.",
                    "inputSchema": { "type": "object", "properties": {} }
                },
                {
                    "name": "mempalace_list_wings",
                    "description": "Returns all wings with counts.",
                    "inputSchema": { "type": "object", "properties": {} }
                },
                {
                    "name": "mempalace_list_rooms",
                    "description": "Returns rooms within a wing.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "wing": { "type": "string" }
                        },
                        "required": ["wing"]
                    }
                },
                {
                    "name": "mempalace_get_taxonomy",
                    "description": "Returns full wing -> room -> count tree.",
                    "inputSchema": { "type": "object", "properties": {} }
                },
                {
                    "name": "mempalace_search",
                    "description": "Semantic search.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "query": { "type": "string" },
                            "wing": { "type": "string" },
                            "room": { "type": "string" },
                            "n_results": { "type": "integer", "default": 5 }
                        },
                        "required": ["query"]
                    }
                },
                {
                    "name": "laputa_semantic_search",
                    "description": "Typed semantic search with similarity scores and optional heat reranking.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "query": { "type": "string" },
                            "wing": { "type": "string" },
                            "room": { "type": "string" },
                            "top_k": { "type": "integer", "default": 10 },
                            "include_discarded": { "type": "boolean", "default": false },
                            "sort_by_heat": { "type": "boolean", "default": false }
                        },
                        "required": ["query"]
                    }
                },
                {
                    "name": "mempalace_check_duplicate",
                    "description": "Similarity check.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "text": { "type": "string" },
                            "threshold": { "type": "number", "default": 0.9 }
                        },
                        "required": ["text"]
                    }
                },
                {
                    "name": "mempalace_get_aaak_spec",
                    "description": "Returns the AAAK spec.",
                    "inputSchema": { "type": "object", "properties": {} }
                },
                {
                    "name": "mempalace_traverse_graph",
                    "description": "Palace graph walk.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "start_room": { "type": "string" },
                            "max_hops": { "type": "integer", "default": 2 }
                        },
                        "required": ["start_room"]
                    }
                },
                {
                    "name": "mempalace_find_tunnels",
                    "description": "Bridge rooms.",
                    "inputSchema": { "type": "object", "properties": {} }
                },
                {
                    "name": "mempalace_graph_stats",
                    "description": "Graph overview.",
                    "inputSchema": { "type": "object", "properties": {} }
                },
                {
                    "name": "mempalace_add_drawer",
                    "description": "File verbatim content.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "content": { "type": "string" },
                            "wing": { "type": "string" },
                            "room": { "type": "string" }
                        },
                        "required": ["content"]
                    }
                },
                {
                    "name": "mempalace_delete_drawer",
                    "description": "Remove drawer.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "memory_id": { "type": "integer" }
                        },
                        "required": ["memory_id"]
                    }
                },
                {
                    "name": "mempalace_kg_query",
                    "description": "KG access.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "entity": { "type": "string" },
                            "direction": { "type": "string", "enum": ["incoming", "outgoing", "both"], "default": "both" }
                        },
                        "required": ["entity"]
                    }
                },
                {
                    "name": "mempalace_kg_add",
                    "description": "Add triple to KG.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "subject": { "type": "string" },
                            "predicate": { "type": "string" },
                            "object": { "type": "string" }
                        },
                        "required": ["subject", "predicate", "object"]
                    }
                },
                {
                    "name": "mempalace_kg_invalidate",
                    "description": "Invalidate triple in KG.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "subject": { "type": "string" },
                            "predicate": { "type": "string" },
                            "object": { "type": "string" }
                        },
                        "required": ["subject", "predicate", "object"]
                    }
                },
                {
                    "name": "mempalace_kg_timeline",
                    "description": "KG timeline.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "entity": { "type": "string" }
                        },
                        "required": ["entity"]
                    }
                },
                {
                    "name": "mempalace_kg_stats",
                    "description": "KG stats.",
                    "inputSchema": { "type": "object", "properties": {} }
                },
                {
                    "name": "mempalace_diary_write",
                    "description": "Agent journal write.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "agent": { "type": "string" },
                            "content": { "type": "string" }
                        },
                        "required": ["agent", "content"]
                    }
                },
                {
                    "name": "mempalace_diary_read",
                    "description": "Agent journal read.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "agent": { "type": "string" },
                            "last_n": { "type": "integer", "default": 5 }
                        },
                        "required": ["agent"]
                    }
                },
                {
                    "name": "mempalace_prune",
                    "description": "Semantic deduplication. Finds and merges similar memories.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "threshold": { "type": "number", "default": 0.85 },
                            "dry_run": { "type": "boolean", "default": true },
                            "wing": { "type": "string" }
                        }
                    }
                }
            ]
        }))
    }

    async fn handle_tools_call(&mut self, params: Option<Value>) -> Result<Value> {
        let params = params.ok_or_else(|| anyhow!("Missing params"))?;
        let name = params["name"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing tool name"))?;
        let args = &params["arguments"];

        let tool_result = match name {
            "laputa_init" => self.laputa_init(args).await,
            "laputa_diary_write" => self.laputa_diary_write(args).await,
            "laputa_recall" => self.laputa_recall(args).await,
            "laputa_wakeup_generate" => self.laputa_wakeup_generate(args).await,
            "laputa_mark_important" => self.laputa_mark_important(args).await,
            "laputa_get_heat_status" => self.laputa_get_heat_status(args).await,
            "mempalace_status" => self.mempalace_status().await,
            "mempalace_list_wings" => self.mempalace_list_wings().await,
            "mempalace_list_rooms" => self.mempalace_list_rooms(args).await,
            "mempalace_get_taxonomy" => self.mempalace_get_taxonomy().await,
            "mempalace_search" => self.mempalace_search(args).await,
            "laputa_semantic_search" => self.laputa_semantic_search(args).await,
            "mempalace_check_duplicate" => self.mempalace_check_duplicate(args).await,
            "mempalace_get_aaak_spec" => self.mempalace_get_aaak_spec().await,
            "mempalace_traverse_graph" => self.mempalace_traverse_graph(args).await,
            "mempalace_find_tunnels" => self.mempalace_find_tunnels().await,
            "mempalace_graph_stats" => self.mempalace_graph_stats().await,
            "mempalace_add_drawer" => self.mempalace_add_drawer(args).await,
            "mempalace_delete_drawer" => self.mempalace_delete_drawer(args).await,
            "mempalace_kg_query" => self.mempalace_kg_query(args).await,
            "mempalace_kg_add" => self.mempalace_kg_add(args).await,
            "mempalace_kg_invalidate" => self.mempalace_kg_invalidate(args).await,
            "mempalace_kg_timeline" => self.mempalace_kg_timeline(args).await,
            "mempalace_kg_stats" => self.mempalace_kg_stats().await,
            "mempalace_diary_write" => self.mempalace_diary_write(args).await,
            "mempalace_diary_read" => self.mempalace_diary_read(args).await,
            "mempalace_prune" => self.mempalace_prune(args).await,
            _ => Err(LaputaError::NotFound(format!("Unknown tool: {name}")).into()),
        }?;

        // Wrap in MCP-compliant content format
        Ok(json!({
            "content": [{
                "type": "text",
                "text": serde_json::to_string(&tool_result)?
            }]
        }))
    }

    pub(crate) async fn laputa_init(&self, args: &Value) -> Result<Value> {
        let user_name = required_string(args, "user_name")?.trim();
        let initializer = IdentityInitializer::new(&self.config.config_dir);
        let db_path = initializer.initialize(user_name)?;

        Ok(json!({
            "status": "initialized",
            "user_name": user_name,
            "db_path": db_path,
            "identity_path": initializer.identity_path.display().to_string()
        }))
    }

    pub(crate) async fn laputa_diary_write(&self, args: &Value) -> Result<Value> {
        self.ensure_initialized()?;

        let diary = Diary::new(self.config.config_dir.join("vectors.db"))?;
        let request = DiaryWriteRequest {
            agent: required_string(args, "agent")?.to_string(),
            content: required_string(args, "content")?.to_string(),
            tags: args["tags"]
                .as_array()
                .map(|items| {
                    items
                        .iter()
                        .filter_map(|item| item.as_str().map(ToString::to_string))
                        .collect::<Vec<_>>()
                })
                .unwrap_or_default(),
            emotion: optional_string(args, "emotion"),
            timestamp: optional_string(args, "timestamp"),
            wing: optional_string(args, "wing"),
            room: optional_string(args, "room"),
        };

        let memory_id = diary.write(request)?;
        Ok(json!({
            "status": "success",
            "memory_id": memory_id
        }))
    }

    pub(crate) async fn laputa_recall(&self, args: &Value) -> Result<Value> {
        self.ensure_initialized()?;

        let time_range = required_string(args, "time_range")?;
        let (start, end) = parse_time_range(time_range)?;
        let mut query = RecallQuery::by_time_range(start, end)
            .with_limit(parse_recall_limit(args, "limit")?)
            .include_discarded(args["include_discarded"].as_bool().unwrap_or(false));

        if let Some(wing) = optional_string(args, "wing") {
            query = query.with_wing(wing);
        }
        if let Some(room) = optional_string(args, "room") {
            query = query.with_room(room);
        }

        let records = self.searcher.recall_by_time_range(query).await?;
        Ok(json!({
            "time_range": time_range,
            "results": records.iter().map(memory_record_json).collect::<Vec<_>>()
        }))
    }

    pub(crate) async fn laputa_wakeup_generate(&self, args: &Value) -> Result<Value> {
        self.ensure_initialized()?;

        let wing = optional_string(args, "wing");
        let wakepack = self.searcher.wake_up(wing.clone()).await?;
        Ok(json!({
            "wing": wing,
            "wakepack": wakepack
        }))
    }

    pub(crate) async fn laputa_mark_important(&self, args: &Value) -> Result<Value> {
        self.ensure_initialized()?;

        let memory_id = parse_memory_id(args, "memory_id")?;
        let reason = optional_string(args, "reason")
            .unwrap_or_else(|| "marked important via MCP".to_string());
        let storage = self.open_vector_storage()?;
        let updated = storage.apply_intervention(
            memory_id,
            UserIntervention::Important {
                reason: reason.clone(),
            },
        )?;

        Ok(json!({
            "status": "success",
            "memory_id": updated.id,
            "heat_i32": updated.heat_i32,
            "last_accessed": updated.last_accessed.to_rfc3339(),
            "access_count": updated.access_count,
            "is_archive_candidate": updated.is_archive_candidate,
            "reason": updated.reason
        }))
    }

    pub(crate) async fn laputa_get_heat_status(&self, args: &Value) -> Result<Value> {
        self.ensure_initialized()?;

        let memory_id = parse_memory_id(args, "memory_id")?;
        let storage = self.open_vector_storage()?;
        let record = storage.get_memory_by_id(memory_id)?;

        Ok(json!({
            "memory_id": record.id,
            "heat_i32": record.heat_i32,
            "last_accessed": record.last_accessed.to_rfc3339(),
            "access_count": record.access_count,
            "is_archive_candidate": record.is_archive_candidate
        }))
    }

    fn ensure_initialized(&self) -> Result<()> {
        let initializer = IdentityInitializer::new(&self.config.config_dir);
        if initializer.is_initialized() {
            return Ok(());
        }

        Err(LaputaError::ConfigError(format!(
            "Laputa is not initialized in {}. Call laputa_init first.",
            self.config.config_dir.display()
        ))
        .into())
    }

    fn open_vector_storage(&self) -> Result<VectorStorage> {
        VectorStorage::new(
            self.config.config_dir.join("vectors.db"),
            self.config.config_dir.join("vectors.usearch"),
        )
        .map_err(map_anyhow_to_laputa_error)
        .map_err(Into::into)
    }

    pub(crate) async fn mempalace_status(&self) -> Result<Value> {
        let count = match VectorStorage::new(
            self.config.config_dir.join("vectors.db"),
            self.config.config_dir.join("vectors.usearch"),
        ) {
            Ok(vs) => vs.memory_count().unwrap_or(0),
            Err(_) => 0,
        };

        Ok(json!({
            "total_memories": count,
            "wings": self.pg.wings.len(),
            "rooms": self.pg.rooms.len(),
            "protocol": "mempalace-mcp-v1",
            "aaak_spec": "3.2",
            "storage_engine": "pure-rust-usearch"
        }))
    }

    pub(crate) async fn mempalace_list_wings(&self) -> Result<Value> {
        let mut wings = HashMap::new();
        for (wing, rooms) in &self.pg.wings {
            wings.insert(wing.clone(), rooms.len());
        }
        Ok(json!({ "wings": wings }))
    }

    pub(crate) async fn mempalace_list_rooms(&self, args: &Value) -> Result<Value> {
        let wing = args["wing"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing wing"))?;
        let rooms = self.pg.wings.get(wing).cloned().unwrap_or_default();
        Ok(json!({ "wing": wing, "rooms": rooms }))
    }

    pub(crate) async fn mempalace_get_taxonomy(&self) -> Result<Value> {
        let mut taxonomy = HashMap::new();
        let max_wings = 100; // Hard limit for safety
        for (i, (wing, rooms)) in self.pg.wings.iter().enumerate() {
            if i >= max_wings {
                break;
            }
            let mut room_counts = HashMap::new();
            for room in rooms {
                room_counts.insert(room.clone(), 0);
            }
            taxonomy.insert(wing.clone(), room_counts);
        }
        Ok(json!({ "taxonomy": taxonomy }))
    }

    pub(crate) async fn mempalace_search(&self, args: &Value) -> Result<Value> {
        let query = args["query"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing query"))?;
        let wing = args["wing"].as_str().map(|s| s.to_string());
        let room = args["room"].as_str().map(|s| s.to_string());
        let n_results = args["n_results"].as_u64().unwrap_or(5) as usize;

        let results = self
            .searcher
            .search_memories(query, wing, room, n_results)
            .await?;
        Ok(results)
    }

    pub(crate) async fn laputa_semantic_search(&self, args: &Value) -> Result<Value> {
        let query = args["query"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing query"))?;
        let top_k = args["top_k"].as_u64().unwrap_or(10) as usize;
        let options = SemanticSearchOptions {
            wing: args["wing"].as_str().map(|value| value.to_string()),
            room: args["room"].as_str().map(|value| value.to_string()),
            include_discarded: args["include_discarded"].as_bool().unwrap_or(false),
            sort_by_heat: args["sort_by_heat"].as_bool().unwrap_or(false),
        };

        let results = self
            .searcher
            .semantic_search(query, top_k, options.clone())
            .await?;
        Ok(Searcher::format_semantic_json_results(
            query, &options, &results,
        ))
    }

    pub(crate) async fn mempalace_check_duplicate(&self, args: &Value) -> Result<Value> {
        let text = args["text"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing text"))?;
        let threshold = args["threshold"].as_f64().unwrap_or(0.9);

        let results = self.searcher.search_memories(text, None, None, 1).await?;
        let mut is_duplicate = false;
        let mut similarity = 0.0;

        if let Some(hits) = results["results"].as_array() {
            if let Some(first) = hits.first() {
                similarity = first["similarity"].as_f64().unwrap_or(0.0);
                if similarity >= threshold {
                    is_duplicate = true;
                }
            }
        }

        Ok(json!({
            "is_duplicate": is_duplicate,
            "similarity": similarity,
            "threshold": threshold
        }))
    }

    pub(crate) async fn mempalace_get_aaak_spec(&self) -> Result<Value> {
        Ok(json!({
            "spec": "AAAK Dialect V:3.2",
            "version": crate::dialect::AAAK_VERSION,
            "compression_ratio": "~30x",
            "layers": ["L0: IDENTITY", "L1: ESSENTIAL", "L2: ON-DEMAND", "L3: SEARCH"],
            "format": "V:3.2\nWING|ROOM|DATE|SOURCE\n0:ENTITIES|TOPICS|\"QUOTE\"|EMOTIONS|FLAGS\nJSON:{overlay}",
            "proposition_format": "V:3.2\nWING|ROOM|DATE|SOURCE\nP0:ENTITIES|TOPICS|EMOTIONS|FLAGS\nP1:ENTITIES|TOPICS",
            "density_range": "1 (compact) – 10 (verbose), default 5",
            "features": [
                "versioning (V:3.2)",
                "adaptive density",
                "metadata overlay (JSON:)",
                "external emotion dictionary (emotions.json)",
                "proposition atomisation (compress_propositions)",
                "faithfulness scoring",
                "delta encoding"
            ],
            "entity_codes": self.dialect.entity_codes.len(),
            "custom_emotions": self.dialect.custom_emotions.len()
        }))
    }

    pub(crate) async fn mempalace_traverse_graph(&self, args: &Value) -> Result<Value> {
        let start_room = args["start_room"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing start_room"))?;
        let max_hops = args["max_hops"].as_u64().unwrap_or(2) as usize;

        let connected = self.pg.find_connected_rooms(start_room, max_hops);
        Ok(json!({ "start_room": start_room, "connected": connected }))
    }

    pub(crate) async fn mempalace_find_tunnels(&self) -> Result<Value> {
        let tunnels = self.pg.find_tunnels();
        Ok(json!({ "tunnels": tunnels }))
    }

    pub(crate) async fn mempalace_graph_stats(&self) -> Result<Value> {
        Ok(json!({
            "total_rooms": self.pg.rooms.len(),
            "total_wings": self.pg.wings.len(),
            "avg_rooms_per_wing": if self.pg.wings.is_empty() { 0.0 } else { self.pg.rooms.len() as f64 / self.pg.wings.len() as f64 }
        }))
    }

    pub(crate) async fn mempalace_add_drawer(&mut self, args: &Value) -> Result<Value> {
        let content = args["content"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing content"))?;
        let wing = args["wing"].as_str().unwrap_or("general");
        let room = args["room"].as_str().unwrap_or("general");

        let memory_id = self.searcher.add_memory(content, wing, room, None, None)?;
        if memory_id <= 0 {
            return Err(anyhow!("Vector storage unavailable; drawer not persisted"));
        }
        self.pg.add_room(room, wing);

        Ok(json!({ "status": "success", "memory_id": memory_id, "wing": wing, "room": room }))
    }

    pub(crate) async fn mempalace_delete_drawer(&self, args: &Value) -> Result<Value> {
        let memory_id = args["memory_id"]
            .as_i64()
            .ok_or_else(|| anyhow!("Missing or invalid memory_id (integer)"))?;
        if memory_id <= 0 {
            return Err(anyhow!("Missing or invalid persisted memory_id"));
        }

        self.searcher.delete_memory(memory_id)?;

        Ok(json!({ "status": "success", "memory_id": memory_id }))
    }

    pub(crate) async fn mempalace_kg_query(&self, args: &Value) -> Result<Value> {
        let entity = args["entity"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing entity"))?;
        let direction = args["direction"].as_str().unwrap_or("both");

        let results = self.kg.query_entity(entity, None, direction)?;
        Ok(json!({ "results": results }))
    }

    pub(crate) async fn mempalace_kg_add(&self, args: &Value) -> Result<Value> {
        let sub = args["subject"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing subject"))?;
        let pred = args["predicate"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing predicate"))?;
        let obj = args["object"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing object"))?;

        let id = self
            .kg
            .add_triple(sub, pred, obj, None, None, 1.0, None, None)?;
        Ok(json!({ "status": "success", "triple_id": id }))
    }

    pub(crate) async fn mempalace_kg_invalidate(&self, args: &Value) -> Result<Value> {
        let sub = args["subject"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing subject"))?;
        let pred = args["predicate"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing predicate"))?;
        let obj = args["object"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing object"))?;

        self.kg.invalidate(sub, pred, obj, None)?;
        Ok(json!({ "status": "success" }))
    }

    pub(crate) async fn mempalace_kg_timeline(&self, args: &Value) -> Result<Value> {
        let entity = args["entity"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing entity"))?;
        let results = self.kg.query_entity(entity, None, "both")?;

        // Simple timeline sort by valid_from
        let mut sorted = results;
        sorted.sort_by(|a, b| {
            let af = a["valid_from"].as_str().unwrap_or("");
            let bf = b["valid_from"].as_str().unwrap_or("");
            af.cmp(bf)
        });

        Ok(json!({ "entity": entity, "timeline": sorted }))
    }

    pub(crate) async fn mempalace_kg_stats(&self) -> Result<Value> {
        let stats = self.kg.stats()?;
        Ok(stats)
    }

    pub(crate) async fn mempalace_diary_write(&self, args: &Value) -> Result<Value> {
        let agent = args["agent"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing agent"))?;
        let content = args["content"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing content"))?;
        let diary_path = self.config.config_dir.join("vectors.db");
        let tags = args["tags"]
            .as_array()
            .map(|items| {
                items
                    .iter()
                    .filter_map(|item| item.as_str().map(ToString::to_string))
                    .collect::<Vec<_>>()
            })
            .unwrap_or_default();
        let diary = Diary::new(&diary_path)?;
        let memory_id = diary.write(DiaryWriteRequest {
            agent: agent.to_string(),
            content: content.to_string(),
            tags,
            emotion: args["emotion"].as_str().map(ToString::to_string),
            timestamp: args["timestamp"].as_str().map(ToString::to_string),
            wing: args["wing"].as_str().map(ToString::to_string),
            room: args["room"].as_str().map(ToString::to_string),
        })?;
        Ok(json!({ "status": "success", "memory_id": memory_id }))
    }

    pub(crate) async fn mempalace_diary_read(&self, args: &Value) -> Result<Value> {
        let agent = args["agent"]
            .as_str()
            .ok_or_else(|| anyhow!("Missing agent"))?;
        let last_n = args["last_n"].as_u64().unwrap_or(5) as usize;
        let diary_path = self.config.config_dir.join("vectors.db");

        let entries = diary::read_diary_at(&diary_path, agent, last_n)?;
        Ok(json!({ "entries": entries }))
    }

    pub(crate) async fn mempalace_prune(&self, args: &Value) -> Result<Value> {
        let threshold = args["threshold"].as_f64().unwrap_or(0.85) as f32;
        let dry_run = args["dry_run"].as_bool().unwrap_or(true);
        let wing = args["wing"].as_str().map(|s| s.to_string());

        let storage_path = self.config.config_dir.join("palace.db");
        let storage = crate::storage::Storage::new(path_to_str(&storage_path)?)?;

        let report = match storage
            .prune_memories(&self.config, threshold, dry_run, wing)
            .await
        {
            Ok(report) => report,
            Err(_) => crate::storage::PruneReport {
                clusters_found: 0,
                merged: 0,
                tokens_saved_est: 0,
            },
        };

        Ok(json!({
            "status": "success",
            "dry_run": dry_run,
            "report": report
        }))
    }
}

fn required_string<'a>(args: &'a Value, field: &str) -> Result<&'a str> {
    args[field]
        .as_str()
        .ok_or_else(|| LaputaError::ValidationError(format!("Missing or invalid {field}")).into())
}

fn optional_string(args: &Value, field: &str) -> Option<String> {
    args[field]
        .as_str()
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToString::to_string)
}

fn parse_recall_limit(args: &Value, field: &str) -> Result<usize> {
    if args[field].is_null() {
        return Ok(100);
    }

    if let Some(value) = args[field].as_i64() {
        let limit = if value <= 0 {
            MIN_RECALL_LIMIT
        } else {
            usize::try_from(value)
                .unwrap_or(MAX_RECALL_LIMIT)
                .clamp(MIN_RECALL_LIMIT, MAX_RECALL_LIMIT)
        };

        return Ok(limit);
    }

    if let Some(value) = args[field].as_u64() {
        return Ok(usize::try_from(value)
            .unwrap_or(MAX_RECALL_LIMIT)
            .clamp(MIN_RECALL_LIMIT, MAX_RECALL_LIMIT));
    }

    Err(LaputaError::ValidationError(format!("{field} must be an integer")).into())
}

fn parse_time_range(raw: &str) -> Result<(i64, i64)> {
    let (start_raw, end_raw) = raw.split_once('~').ok_or_else(|| {
        LaputaError::ValidationError(format!(
            "time_range must use `YYYY-MM-DD~YYYY-MM-DD`, got `{raw}`"
        ))
    })?;

    let start_date = NaiveDate::parse_from_str(start_raw.trim(), "%Y-%m-%d").map_err(|_| {
        LaputaError::ValidationError(format!(
            "invalid start date in time_range `{raw}`; expected YYYY-MM-DD"
        ))
    })?;
    let end_date = NaiveDate::parse_from_str(end_raw.trim(), "%Y-%m-%d").map_err(|_| {
        LaputaError::ValidationError(format!(
            "invalid end date in time_range `{raw}`; expected YYYY-MM-DD"
        ))
    })?;

    validate_time_range_dates(start_date, end_date, raw)?;

    let start = Utc
        .from_utc_datetime(
            &start_date
                .and_hms_opt(0, 0, 0)
                .ok_or_else(|| LaputaError::ValidationError("invalid start date".to_string()))?,
        )
        .timestamp();
    let end = Utc
        .from_utc_datetime(
            &end_date
                .and_hms_opt(23, 59, 59)
                .ok_or_else(|| LaputaError::ValidationError("invalid end date".to_string()))?,
        )
        .timestamp();

    if start > end {
        return Err(LaputaError::ValidationError(format!(
            "start date must be before or equal to end date in `{raw}`"
        ))
        .into());
    }

    Ok((start, end))
}

fn parse_memory_id(args: &Value, field: &str) -> Result<i64> {
    if let Some(id) = args[field].as_i64() {
        if id > 0 {
            return Ok(id);
        }
        return Err(LaputaError::ValidationError(format!(
            "invalid {field}; expected positive numeric memory_id"
        ))
        .into());
    }

    if let Some(raw) = args[field].as_str() {
        let trimmed = raw.trim();
        if let Ok(id) = trimmed.parse::<i64>() {
            if id > 0 {
                return Ok(id);
            }
            return Err(LaputaError::ValidationError(format!(
                "invalid {field}; expected positive numeric memory_id"
            ))
            .into());
        }

        if Uuid::parse_str(trimmed).is_ok() {
            return Err(LaputaError::ValidationError(
                "Phase 1 MCP currently accepts numeric memory_id values; UUID support is not wired yet."
                    .to_string(),
            )
            .into());
        }
    }

    Err(LaputaError::ValidationError(format!("invalid {field}; expected numeric memory_id")).into())
}

fn validate_time_range_dates(start_date: NaiveDate, end_date: NaiveDate, raw: &str) -> Result<()> {
    validate_date_year(start_date, "start", raw)?;
    validate_date_year(end_date, "end", raw)?;

    let span_days = end_date.signed_duration_since(start_date).num_days();
    if span_days > MAX_TIME_RANGE_DAYS {
        return Err(LaputaError::ValidationError(format!(
            "time_range `{raw}` exceeds the maximum span of {MAX_TIME_RANGE_DAYS} days"
        ))
        .into());
    }

    Ok(())
}

fn validate_date_year(date: NaiveDate, label: &str, raw: &str) -> Result<()> {
    if !(MIN_ALLOWED_DATE_YEAR..=MAX_ALLOWED_DATE_YEAR).contains(&date.year()) {
        return Err(LaputaError::ValidationError(format!(
            "{label} date in time_range `{raw}` must stay within {MIN_ALLOWED_DATE_YEAR:04}-01-01 and {MAX_ALLOWED_DATE_YEAR:04}-12-31"
        ))
        .into());
    }

    Ok(())
}

fn path_to_str(path: &std::path::Path) -> Result<&str> {
    path.to_str().ok_or_else(|| {
        LaputaError::InvalidPath(format!("path is not valid UTF-8: {}", path.display())).into()
    })
}

fn open_knowledge_graph(path: &std::path::Path) -> Result<KnowledgeGraph> {
    Ok(KnowledgeGraph::new(path_to_str(path)?)?)
}

fn memory_record_json(record: &crate::vector_storage::MemoryRecord) -> Value {
    let text = visible_memory_text(&record.text_content);
    json!({
        "memory_id": record.id,
        "text": text,
        "wing": &record.wing,
        "room": &record.room,
        "source_file": &record.source_file,
        "valid_from": record.valid_from,
        "valid_to": record.valid_to,
        "heat_i32": record.heat_i32,
        "last_accessed": record.last_accessed.to_rfc3339(),
        "access_count": record.access_count,
        "is_archive_candidate": record.is_archive_candidate,
        "discard_candidate": record.discard_candidate,
        "reason": &record.reason,
    })
}

fn visible_memory_text(text: &str) -> &str {
    const DIARY_META_PREFIX: &str = "DIARY_META:";
    if text.starts_with(DIARY_META_PREFIX) {
        if let Some((_, content)) = text.split_once('\n') {
            return content;
        }
    }
    text
}

fn map_anyhow_to_laputa_error(error: anyhow::Error) -> LaputaError {
    if let Some(laputa_error) = error.downcast_ref::<LaputaError>() {
        return laputa_error.clone();
    }

    if let Some(io_error) = error.downcast_ref::<std::io::Error>() {
        return LaputaError::from(std::io::Error::new(io_error.kind(), io_error.to_string()));
    }

    if let Some(sql_error) = error.downcast_ref::<rusqlite::Error>() {
        return LaputaError::StorageError(sql_error.to_string());
    }

    let message = error.to_string();
    if message.contains("Memory not found") {
        return LaputaError::NotFound(message);
    }
    if message.contains("Cannot open SQLite")
        || message.contains("Non-UTF8 index path")
        || message.contains("Vector storage unavailable")
    {
        return LaputaError::ConfigError(message);
    }
    if message.contains("Invalid RFC3339")
        || message.contains("Unknown emotion")
        || message.contains("cannot be empty")
    {
        return LaputaError::ValidationError(message);
    }

    LaputaError::StorageError(message)
}

fn map_jsonrpc_error(error: &anyhow::Error) -> (i32, String) {
    if let Some(laputa_error) = error.downcast_ref::<LaputaError>() {
        let code = match laputa_error {
            LaputaError::ValidationError(_) => -32602,
            LaputaError::NotFound(_) => -32004,
            LaputaError::ConfigError(_) | LaputaError::AlreadyInitialized(_) => -32001,
            LaputaError::StorageError(_)
            | LaputaError::HeatThresholdError(_)
            | LaputaError::ArchiveError(_)
            | LaputaError::WakepackSizeExceeded(_)
            | LaputaError::InvalidPath(_) => -32603,
        };
        return (code, laputa_error.to_string());
    }

    let message = error.to_string();
    if message.contains("Missing ") || message.contains("invalid ") {
        return (-32602, message);
    }

    (-32603, message)
}

pub async fn run_mcp_server() -> Result<()> {
    let config = MempalaceConfig::default();
    let mut server = McpServer::new(config).await?;
    server.run().await
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::cli::commands::{
        Cli, Commands, DiaryCommand, DiarySubcommands, DiaryWriteCommand, InitCommand,
    };
    use crate::cli::handlers as cli_handlers;

    fn setup_test() -> (MempalaceConfig, tempfile::TempDir) {
        let temp_dir = tempfile::tempdir().unwrap();
        let config = MempalaceConfig::new(Some(temp_dir.path().to_path_buf()));
        (config, temp_dir)
    }

    fn make_request(method: &str, params: Option<Value>, id: Option<Value>) -> JsonRpcRequest {
        JsonRpcRequest {
            jsonrpc: "2.0".to_string(),
            method: method.to_string(),
            params,
            id,
        }
    }

    fn parse_content_text(result: &Value) -> Value {
        let content = result["content"].as_array().expect("missing content array");
        let text = content[0]["text"].as_str().expect("missing text field");
        serde_json::from_str(text).expect("content text should be valid JSON")
    }

    // ── Protocol-level tests ─────────────────────────────────────────

    #[tokio::test]
    async fn test_handle_request_initialize() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request("initialize", None, Some(json!(1)));
        let resp = server.handle_request(req).await;

        assert!(resp.error.is_none());
        let res = resp.result.unwrap();
        assert_eq!(res["protocolVersion"], "2024-11-05");
        assert_eq!(res["serverInfo"]["name"], "laputa");
        assert!(res["capabilities"]["tools"].is_object());
        // resources and prompts should NOT be advertised
        assert!(res["capabilities"]["resources"].is_null());
        assert!(res["capabilities"]["prompts"].is_null());
    }

    #[tokio::test]
    async fn test_handle_request_tools_list() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request("tools/list", None, Some(json!(2)));
        let resp = server.handle_request(req).await;

        assert!(resp.error.is_none());
        let res = resp.result.unwrap();
        let tools = res["tools"].as_array().unwrap();
        assert!(
            tools.len() >= 26,
            "Expected at least 26 tools, got {}",
            tools.len()
        );
    }

    #[tokio::test]
    async fn test_handle_request_tools_call_content_wrapper() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request(
            "tools/call",
            Some(json!({ "name": "mempalace_status", "arguments": {} })),
            Some(json!(3)),
        );
        let resp = server.handle_request(req).await;

        assert!(resp.error.is_none());
        let res = resp.result.unwrap();
        // Must have MCP-compliant content wrapper
        let content = res["content"].as_array().expect("missing content array");
        assert!(!content.is_empty());
        assert_eq!(content[0]["type"], "text");
        // text field must be valid JSON
        let inner: Value = serde_json::from_str(content[0]["text"].as_str().unwrap())
            .expect("text not valid JSON");
        assert!(inner["total_memories"].is_number());
        assert!(inner["protocol"].is_string());
    }

    #[tokio::test]
    async fn test_handle_request_resources_list() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request("resources/list", None, Some(json!(4)));
        let resp = server.handle_request(req).await;

        assert!(resp.error.is_none());
        let res = resp.result.unwrap();
        assert_eq!(res["resources"].as_array().unwrap().len(), 0);
    }

    #[tokio::test]
    async fn test_handle_request_resources_read_error() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request("resources/read", None, Some(json!(5)));
        let resp = server.handle_request(req).await;

        assert!(resp.error.is_some());
        assert!(resp.error.unwrap().message.contains("Resource not found"));
    }

    #[tokio::test]
    async fn test_handle_request_prompts_list() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request("prompts/list", None, Some(json!(6)));
        let resp = server.handle_request(req).await;

        assert!(resp.error.is_none());
        let res = resp.result.unwrap();
        assert_eq!(res["prompts"].as_array().unwrap().len(), 0);
    }

    #[tokio::test]
    async fn test_handle_request_unknown_method() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request("nonexistent/method", None, Some(json!(7)));
        let resp = server.handle_request(req).await;

        assert!(resp.error.is_none());
        let res = resp.result.unwrap();
        assert!(res.is_object());
    }

    #[tokio::test]
    async fn test_handle_request_preserves_id() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request("initialize", None, Some(json!("my-string-id")));
        let resp = server.handle_request(req).await;
        assert_eq!(resp.id, Some(json!("my-string-id")));
    }

    #[tokio::test]
    async fn test_handle_request_jsonrpc_version() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request("initialize", None, Some(json!(1)));
        let resp = server.handle_request(req).await;
        assert_eq!(resp.jsonrpc, "2.0");
    }

    // ── Tool schema validation ───────────────────────────────────────

    #[tokio::test]
    async fn test_tools_list_schema_completeness() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.handle_tools_list().unwrap();
        let tools = res["tools"].as_array().unwrap();

        for tool in tools {
            let name = tool["name"].as_str().expect("tool missing name");
            assert!(
                tool["description"].as_str().is_some(),
                "tool {} missing description",
                name
            );
            assert!(
                tool["inputSchema"].is_object(),
                "tool {} missing inputSchema",
                name
            );
            assert_eq!(
                tool["inputSchema"]["type"], "object",
                "tool {} inputSchema.type must be 'object'",
                name
            );
        }
    }

    #[tokio::test]
    async fn test_tools_list_expected_names() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.handle_tools_list().unwrap();
        let tools = res["tools"].as_array().unwrap();
        let names: Vec<&str> = tools.iter().map(|t| t["name"].as_str().unwrap()).collect();

        let expected = [
            "laputa_init",
            "laputa_diary_write",
            "laputa_recall",
            "laputa_wakeup_generate",
            "laputa_mark_important",
            "laputa_get_heat_status",
            "mempalace_status",
            "mempalace_list_wings",
            "mempalace_list_rooms",
            "mempalace_get_taxonomy",
            "mempalace_search",
            "mempalace_check_duplicate",
            "mempalace_get_aaak_spec",
            "mempalace_traverse_graph",
            "mempalace_find_tunnels",
            "mempalace_graph_stats",
            "mempalace_add_drawer",
            "mempalace_delete_drawer",
            "mempalace_kg_query",
            "mempalace_kg_add",
            "mempalace_kg_invalidate",
            "mempalace_kg_timeline",
            "mempalace_kg_stats",
            "mempalace_diary_write",
            "mempalace_diary_read",
            "mempalace_prune",
        ];
        for name in &expected {
            assert!(names.contains(name), "missing tool: {}", name);
        }
    }

    #[tokio::test]
    async fn test_laputa_tools_use_snake_case_schema_fields() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.handle_tools_list().unwrap();
        let tools = res["tools"].as_array().unwrap();

        for tool in tools {
            let name = tool["name"].as_str().unwrap();
            if !name.starts_with("laputa_") {
                continue;
            }

            let properties = tool["inputSchema"]["properties"]
                .as_object()
                .expect("properties should be an object");

            for key in properties.keys() {
                assert!(
                    key.chars()
                        .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == '_'),
                    "tool {} has non-snake_case field {}",
                    name,
                    key
                );
            }
        }
    }

    // ── Error / edge-case tests ──────────────────────────────────────

    #[tokio::test]
    async fn test_tools_call_missing_params() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request("tools/call", None, Some(json!(10)));
        let resp = server.handle_request(req).await;
        assert!(resp.error.is_some(), "expected error for missing params");
    }

    #[tokio::test]
    async fn test_tools_call_missing_tool_name() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request(
            "tools/call",
            Some(json!({ "arguments": {} })),
            Some(json!(11)),
        );
        let resp = server.handle_request(req).await;
        assert!(resp.error.is_some(), "expected error for missing tool name");
    }

    #[tokio::test]
    async fn test_tools_call_unknown_tool() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request(
            "tools/call",
            Some(json!({ "name": "nonexistent_tool", "arguments": {} })),
            Some(json!(12)),
        );
        let resp = server.handle_request(req).await;
        assert!(resp.error.is_some());
        assert!(resp.error.unwrap().message.contains("Unknown tool"));
    }

    #[tokio::test]
    async fn test_tools_call_laputa_init_success() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request(
            "tools/call",
            Some(json!({ "name": "laputa_init", "arguments": { "user_name": "Tester" } })),
            Some(json!(13)),
        );
        let resp = server.handle_request(req).await;

        assert!(resp.error.is_none(), "unexpected error: {:?}", resp.error);
        let inner = parse_content_text(&resp.result.unwrap());
        assert_eq!(inner["status"], "initialized");
        assert_eq!(inner["user_name"], "Tester");
        assert!(inner["db_path"].as_str().unwrap().ends_with("laputa.db"));
    }

    #[tokio::test]
    async fn test_tools_call_laputa_init_trims_user_name() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config.clone());
        let req = make_request(
            "tools/call",
            Some(json!({ "name": "laputa_init", "arguments": { "user_name": "  Tester  " } })),
            Some(json!(13_1)),
        );
        let resp = server.handle_request(req).await;

        assert!(resp.error.is_none(), "unexpected error: {:?}", resp.error);
        let inner = parse_content_text(&resp.result.unwrap());
        assert_eq!(inner["user_name"], "Tester");

        let identity = std::fs::read_to_string(config.config_dir.join("identity.md")).unwrap();
        assert!(identity.contains("user_name: Tester\n"));
        assert!(!identity.contains("user_name:   Tester  "));
    }

    #[tokio::test]
    async fn test_tools_call_laputa_init_rejects_blank_user_name() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request(
            "tools/call",
            Some(json!({ "name": "laputa_init", "arguments": { "user_name": "   " } })),
            Some(json!(13_2)),
        );
        let resp = server.handle_request(req).await;

        let error = resp.error.expect("expected validation error");
        assert_eq!(error.code, -32602);
        assert!(error.message.contains("user_name"));
    }

    #[tokio::test]
    async fn test_tools_call_laputa_diary_write_requires_init() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let req = make_request(
            "tools/call",
            Some(json!({
                "name": "laputa_diary_write",
                "arguments": { "agent": "tester", "content": "hello" }
            })),
            Some(json!(14)),
        );
        let resp = server.handle_request(req).await;

        let error = resp.error.expect("expected initialization error");
        assert_eq!(error.code, -32001);
        assert!(error.message.contains("laputa_init"));
    }

    #[tokio::test]
    async fn test_tools_call_laputa_diary_write_and_recall_success() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);

        let init_resp = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({ "name": "laputa_init", "arguments": { "user_name": "Tester" } })),
                Some(json!(15)),
            ))
            .await;
        assert!(init_resp.error.is_none());

        let write_resp = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({
                    "name": "laputa_diary_write",
                    "arguments": {
                        "agent": "tester",
                        "content": "remember this entry",
                        "timestamp": "2026-04-14T08:30:00Z",
                        "tags": ["focus"]
                    }
                })),
                Some(json!(16)),
            ))
            .await;
        assert!(
            write_resp.error.is_none(),
            "write error: {:?}",
            write_resp.error
        );

        let recall_resp = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({
                    "name": "laputa_recall",
                    "arguments": {
                        "time_range": "2026-04-14~2026-04-14",
                        "limit": 10
                    }
                })),
                Some(json!(17)),
            ))
            .await;

        assert!(
            recall_resp.error.is_none(),
            "recall error: {:?}",
            recall_resp.error
        );
        let inner = parse_content_text(&recall_resp.result.unwrap());
        let results = inner["results"].as_array().unwrap();
        assert_eq!(results.len(), 1);
        assert_eq!(results[0]["text"], "remember this entry");
        assert_eq!(results[0]["memory_id"], 1);
    }

    #[tokio::test]
    async fn test_tools_call_laputa_recall_rejects_bad_time_range() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let _ = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({ "name": "laputa_init", "arguments": { "user_name": "Tester" } })),
                Some(json!(18)),
            ))
            .await;

        let resp = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({
                    "name": "laputa_recall",
                    "arguments": { "time_range": "2026/04/14" }
                })),
                Some(json!(19)),
            ))
            .await;

        let error = resp.error.expect("expected validation error");
        assert_eq!(error.code, -32602);
        assert!(error.message.contains("time_range"));
    }

    #[tokio::test]
    async fn test_tools_call_laputa_recall_rejects_extreme_dates() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let _ = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({ "name": "laputa_init", "arguments": { "user_name": "Tester" } })),
                Some(json!(19_1)),
            ))
            .await;

        let resp = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({
                    "name": "laputa_recall",
                    "arguments": { "time_range": "0001-01-01~2026-04-14" }
                })),
                Some(json!(19_2)),
            ))
            .await;

        let error = resp.error.expect("expected validation error");
        assert_eq!(error.code, -32602);
        assert!(error.message.contains("must stay within"));
    }

    #[tokio::test]
    async fn test_tools_call_laputa_recall_rejects_ranges_over_365_days() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let _ = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({ "name": "laputa_init", "arguments": { "user_name": "Tester" } })),
                Some(json!(19_3)),
            ))
            .await;

        let resp = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({
                    "name": "laputa_recall",
                    "arguments": { "time_range": "2025-01-01~2026-04-02" }
                })),
                Some(json!(19_4)),
            ))
            .await;

        let error = resp.error.expect("expected validation error");
        assert_eq!(error.code, -32602);
        assert!(error.message.contains("maximum span"));
    }

    #[tokio::test]
    async fn test_tools_call_laputa_mark_important_and_heat_status_success() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);

        let _ = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({ "name": "laputa_init", "arguments": { "user_name": "Tester" } })),
                Some(json!(20)),
            ))
            .await;

        let write_resp = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({
                    "name": "laputa_diary_write",
                    "arguments": {
                        "agent": "tester",
                        "content": "important memory",
                        "timestamp": "2026-04-14T09:00:00Z"
                    }
                })),
                Some(json!(21)),
            ))
            .await;
        assert!(write_resp.error.is_none());
        let memory_id = parse_content_text(&write_resp.result.unwrap())["memory_id"]
            .as_i64()
            .unwrap();

        let mark_resp = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({
                    "name": "laputa_mark_important",
                    "arguments": {
                        "memory_id": memory_id,
                        "reason": "pin for wakeup"
                    }
                })),
                Some(json!(22)),
            ))
            .await;
        assert!(
            mark_resp.error.is_none(),
            "mark error: {:?}",
            mark_resp.error
        );
        let mark_inner = parse_content_text(&mark_resp.result.unwrap());
        assert_eq!(mark_inner["heat_i32"], 9000);
        assert_eq!(mark_inner["is_archive_candidate"], false);

        let heat_resp = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({
                    "name": "laputa_get_heat_status",
                    "arguments": { "memory_id": memory_id }
                })),
                Some(json!(23)),
            ))
            .await;
        assert!(
            heat_resp.error.is_none(),
            "heat error: {:?}",
            heat_resp.error
        );
        let heat_inner = parse_content_text(&heat_resp.result.unwrap());
        assert_eq!(heat_inner["memory_id"], memory_id);
        assert_eq!(heat_inner["heat_i32"], 9000);
        assert_eq!(heat_inner["is_archive_candidate"], false);
    }

    #[tokio::test]
    async fn test_tools_call_laputa_mark_important_rejects_uuid_memory_id() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let _ = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({ "name": "laputa_init", "arguments": { "user_name": "Tester" } })),
                Some(json!(24)),
            ))
            .await;

        let resp = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({
                    "name": "laputa_mark_important",
                    "arguments": {
                        "memory_id": "550e8400-e29b-41d4-a716-446655440000"
                    }
                })),
                Some(json!(25)),
            ))
            .await;

        let error = resp.error.expect("expected validation error");
        assert_eq!(error.code, -32602);
        assert!(error.message.contains("numeric memory_id"));
    }

    #[tokio::test]
    async fn test_tools_call_laputa_mark_important_rejects_non_positive_memory_id() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let _ = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({ "name": "laputa_init", "arguments": { "user_name": "Tester" } })),
                Some(json!(25_1)),
            ))
            .await;

        for memory_id in [json!(0), json!(-1)] {
            let resp = server
                .handle_request(make_request(
                    "tools/call",
                    Some(json!({
                        "name": "laputa_mark_important",
                        "arguments": { "memory_id": memory_id }
                    })),
                    Some(json!(25_2)),
                ))
                .await;

            let error = resp.error.expect("expected validation error");
            assert_eq!(error.code, -32602);
            assert!(error.message.contains("positive numeric memory_id"));
        }
    }

    #[tokio::test]
    async fn test_laputa_recall_limit_is_clamped_for_zero_and_large_values() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config.clone());

        assert_eq!(
            parse_recall_limit(&json!({ "limit": 0 }), "limit").unwrap(),
            1
        );
        assert_eq!(
            parse_recall_limit(&json!({ "limit": -5 }), "limit").unwrap(),
            1
        );
        assert_eq!(
            parse_recall_limit(&json!({ "limit": 50_000 }), "limit").unwrap(),
            10_000
        );
        assert_eq!(parse_recall_limit(&json!({}), "limit").unwrap(), 100);

        let _ = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({ "name": "laputa_init", "arguments": { "user_name": "Tester" } })),
                Some(json!(25_3)),
            ))
            .await;

        for i in 0..3 {
            let response = server
                .laputa_diary_write(&json!({
                    "agent": "tester",
                    "content": format!("important retained journal entry number {i}"),
                    "timestamp": format!("2026-04-14T08:0{i}:00Z"),
                }))
                .await
                .unwrap();
            assert!(response["memory_id"].as_i64().unwrap() > 0);
        }

        let clamped_min = server
            .laputa_recall(&json!({
                "time_range": "2026-04-14~2026-04-14",
                "limit": 0,
                "include_discarded": true
            }))
            .await
            .unwrap();
        assert_eq!(clamped_min["results"].as_array().unwrap().len(), 1);

        let clamped_negative = server
            .laputa_recall(&json!({
                "time_range": "2026-04-14~2026-04-14",
                "limit": -5,
                "include_discarded": true
            }))
            .await
            .unwrap();
        assert_eq!(clamped_negative["results"].as_array().unwrap().len(), 1);

        let clamped_max = server
            .laputa_recall(&json!({
                "time_range": "2026-04-14~2026-04-14",
                "limit": 50_000,
                "include_discarded": true
            }))
            .await
            .unwrap();
        assert!(!clamped_max["results"].as_array().unwrap().is_empty());

        let conn = rusqlite::Connection::open(config.config_dir.join("vectors.db")).unwrap();
        let stored: i64 = conn
            .query_row("SELECT COUNT(*) FROM memories", [], |row| row.get(0))
            .unwrap();
        assert!(stored >= 1);
    }

    #[tokio::test]
    async fn test_laputa_recall_limit_rejects_non_integer_values() {
        assert!(parse_recall_limit(&json!({ "limit": 50.5 }), "limit").is_err());
        assert!(parse_recall_limit(&json!({ "limit": "hello" }), "limit").is_err());

        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let _ = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({ "name": "laputa_init", "arguments": { "user_name": "Tester" } })),
                Some(json!(25_4)),
            ))
            .await;

        let resp = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({
                    "name": "laputa_recall",
                    "arguments": {
                        "time_range": "2026-04-14~2026-04-14",
                        "limit": "hello"
                    }
                })),
                Some(json!(25_5)),
            ))
            .await;

        let error = resp.error.expect("expected validation error");
        assert_eq!(error.code, -32602);
        assert!(error.message.contains("limit"));
    }

    #[tokio::test]
    async fn test_tools_call_laputa_wakeup_generate_success() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let _ = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({ "name": "laputa_init", "arguments": { "user_name": "Tester" } })),
                Some(json!(26)),
            ))
            .await;

        let resp = server
            .handle_request(make_request(
                "tools/call",
                Some(json!({
                    "name": "laputa_wakeup_generate",
                    "arguments": {}
                })),
                Some(json!(27)),
            ))
            .await;

        assert!(resp.error.is_none(), "wakeup error: {:?}", resp.error);
        let inner = parse_content_text(&resp.result.unwrap());
        assert!(inner["wakepack"].is_string());
    }

    #[tokio::test]
    async fn test_list_rooms_missing_wing() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_list_rooms(&json!({})).await;
        assert!(res.is_err());
    }

    #[tokio::test]
    async fn test_kg_add_missing_fields() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);

        // missing subject
        assert!(server
            .mempalace_kg_add(&json!({"predicate": "is", "object": "x"}))
            .await
            .is_err());
        // missing predicate
        assert!(server
            .mempalace_kg_add(&json!({"subject": "x", "object": "y"}))
            .await
            .is_err());
        // missing object
        assert!(server
            .mempalace_kg_add(&json!({"subject": "x", "predicate": "is"}))
            .await
            .is_err());
    }

    #[tokio::test]
    async fn test_delete_drawer_invalid_id() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        // string instead of integer
        let res = server
            .mempalace_delete_drawer(&json!({"memory_id": "bad"}))
            .await;
        assert!(res.is_err());
    }

    #[tokio::test]
    async fn test_search_missing_query() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_search(&json!({})).await;
        assert!(res.is_err());
    }

    #[tokio::test]
    async fn test_check_duplicate_missing_text() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_check_duplicate(&json!({})).await;
        assert!(res.is_err());
    }

    #[tokio::test]
    async fn test_traverse_graph_missing_start_room() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_traverse_graph(&json!({})).await;
        assert!(res.is_err());
    }

    #[tokio::test]
    async fn test_kg_query_missing_entity() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_kg_query(&json!({})).await;
        assert!(res.is_err());
    }

    #[tokio::test]
    async fn test_kg_timeline_missing_entity() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_kg_timeline(&json!({})).await;
        assert!(res.is_err());
    }

    #[tokio::test]
    async fn test_diary_write_missing_agent() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server
            .mempalace_diary_write(&json!({"content": "hello"}))
            .await;
        assert!(res.is_err());
    }

    #[tokio::test]
    async fn test_diary_write_missing_content() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server
            .mempalace_diary_write(&json!({"agent": "test"}))
            .await;
        assert!(res.is_err());
    }

    #[tokio::test]
    async fn test_diary_read_missing_agent() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_diary_read(&json!({})).await;
        assert!(res.is_err());
    }

    // ── Individual tool tests ────────────────────────────────────────

    #[tokio::test]
    async fn test_mcp_initialize() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.handle_initialize(None).unwrap();
        assert_eq!(res["serverInfo"]["name"], "laputa");
    }

    #[tokio::test]
    async fn test_mcp_tools_list() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.handle_tools_list().unwrap();
        let tools = res["tools"].as_array().unwrap();
        assert!(tools.len() > 10);
    }

    #[tokio::test]
    async fn test_mempalace_status() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_status().await.unwrap();
        assert!(res["total_memories"].is_number());
        assert_eq!(res["protocol"], "mempalace-mcp-v1");
        assert_eq!(res["storage_engine"], "pure-rust-usearch");
        assert_eq!(res["aaak_spec"], "3.2");
    }

    #[tokio::test]
    async fn test_mempalace_list_wings() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        server.pg.add_room("room1", "wing1");

        let res = server.mempalace_list_wings().await.unwrap();
        assert_eq!(res["wings"]["wing1"], 1);
    }

    #[tokio::test]
    async fn test_mempalace_list_wings_empty() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_list_wings().await.unwrap();
        assert_eq!(res["wings"].as_object().unwrap().len(), 0);
    }

    #[tokio::test]
    async fn test_mempalace_list_rooms() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        server.pg.add_room("room1", "wing1");
        server.pg.add_room("room2", "wing1");

        let args = json!({ "wing": "wing1" });
        let res = server.mempalace_list_rooms(&args).await.unwrap();
        let rooms = res["rooms"].as_array().unwrap();
        assert_eq!(rooms.len(), 2);
        assert_eq!(res["wing"], "wing1");
    }

    #[tokio::test]
    async fn test_mempalace_list_rooms_nonexistent_wing() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let args = json!({ "wing": "no_such_wing" });
        let res = server.mempalace_list_rooms(&args).await.unwrap();
        assert_eq!(res["rooms"].as_array().unwrap().len(), 0);
    }

    #[tokio::test]
    async fn test_mempalace_get_taxonomy() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        server.pg.add_room("room1", "wing1");
        server.pg.add_room("room2", "wing2");

        let res = server.mempalace_get_taxonomy().await.unwrap();
        assert!(res["taxonomy"]["wing1"].is_object());
        assert!(res["taxonomy"]["wing2"].is_object());
    }

    #[tokio::test]
    async fn test_mempalace_get_taxonomy_empty() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_get_taxonomy().await.unwrap();
        assert_eq!(res["taxonomy"].as_object().unwrap().len(), 0);
    }

    #[tokio::test]
    async fn test_mempalace_graph_stats() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        server.pg.add_room("room1", "wing1");

        let res = server.mempalace_graph_stats().await.unwrap();
        assert_eq!(res["total_rooms"], 1);
        assert_eq!(res["total_wings"], 1);
        assert!(res["avg_rooms_per_wing"].as_f64().unwrap() > 0.0);
    }

    #[tokio::test]
    async fn test_mempalace_graph_stats_empty() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_graph_stats().await.unwrap();
        assert_eq!(res["total_rooms"], 0);
        assert_eq!(res["total_wings"], 0);
        assert_eq!(res["avg_rooms_per_wing"], 0.0);
    }

    #[tokio::test]
    async fn test_mempalace_get_aaak_spec() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_get_aaak_spec().await.unwrap();
        assert!(res["spec"].as_str().unwrap().contains("AAAK Dialect"));
        assert!(res["version"].is_string());
        assert_eq!(res["compression_ratio"], "~30x");
        assert!(res["layers"].as_array().unwrap().len() == 4);
        assert!(!res["features"].as_array().unwrap().is_empty());
    }

    #[tokio::test]
    async fn test_mempalace_search_empty_palace() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let args = json!({ "query": "hello world" });
        let res = server.mempalace_search(&args).await.unwrap();
        assert!(res["results"].is_array());
    }

    #[tokio::test]
    async fn test_mempalace_search_with_filters() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let args = json!({ "query": "test", "wing": "tech", "room": "code", "n_results": 3 });
        let res = server.mempalace_search(&args).await.unwrap();
        assert!(res["results"].is_array());
    }

    #[tokio::test]
    async fn test_laputa_semantic_search_empty_palace() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let args = json!({ "query": "hello world", "top_k": 3, "sort_by_heat": true });
        let res = server.laputa_semantic_search(&args).await.unwrap();
        assert!(res["results"].is_array());
        assert_eq!(res["filters"]["sort_by_heat"], true);
    }

    #[tokio::test]
    async fn test_mempalace_check_duplicate_empty_palace() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let args = json!({ "text": "something unique", "threshold": 0.95 });
        let res = server.mempalace_check_duplicate(&args).await.unwrap();
        assert_eq!(res["is_duplicate"], false);
        assert!(res["threshold"].as_f64().unwrap() > 0.0);
    }

    #[tokio::test]
    async fn test_mempalace_traverse_graph() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        server.pg.add_room("room1", "wing1");

        let args = json!({ "start_room": "room1", "max_hops": 2 });
        let res = server.mempalace_traverse_graph(&args).await.unwrap();
        assert_eq!(res["start_room"], "room1");
        assert!(res["connected"].is_array());
    }

    #[tokio::test]
    async fn test_mempalace_traverse_graph_default_hops() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let args = json!({ "start_room": "unknown_room" });
        let res = server.mempalace_traverse_graph(&args).await.unwrap();
        assert_eq!(res["start_room"], "unknown_room");
    }

    #[tokio::test]
    async fn test_mempalace_find_tunnels() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_find_tunnels().await.unwrap();
        assert!(res["tunnels"].is_array());
    }

    #[tokio::test]
    async fn test_mempalace_add_drawer() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let args = json!({ "content": "test memory content", "wing": "tech", "room": "rust" });
        match server.mempalace_add_drawer(&args).await {
            Ok(res) => {
                assert_eq!(res["status"], "success");
                assert!(res["memory_id"].as_i64().unwrap_or_default() > 0);
                assert_eq!(res["wing"], "tech");
                assert_eq!(res["room"], "rust");
            }
            Err(err) => {
                assert!(err.to_string().contains("Vector storage unavailable"));
            }
        }
    }

    #[tokio::test]
    async fn test_mempalace_add_drawer_defaults() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);
        let args = json!({ "content": "test memory" });
        match server.mempalace_add_drawer(&args).await {
            Ok(res) => {
                assert_eq!(res["status"], "success");
                assert_eq!(res["wing"], "general");
                assert_eq!(res["room"], "general");
                assert!(res["memory_id"].as_i64().unwrap_or_default() > 0);
            }
            Err(err) => {
                assert!(err.to_string().contains("Vector storage unavailable"));
            }
        }
    }

    #[tokio::test]
    async fn test_mempalace_add_and_delete_drawer() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);

        // Add
        let add_args = json!({ "content": "ephemeral memory" });
        let add_res = match server.mempalace_add_drawer(&add_args).await {
            Ok(res) => res,
            Err(err) => {
                assert!(err.to_string().contains("Vector storage unavailable"));
                return;
            }
        };
        let memory_id = add_res["memory_id"].as_i64().unwrap();

        // Delete
        let del_args = json!({ "memory_id": memory_id });
        let del_res = server.mempalace_delete_drawer(&del_args).await.unwrap();
        assert_eq!(del_res["status"], "success");
        assert_eq!(del_res["memory_id"], memory_id);
    }

    // ── Knowledge Graph tests ────────────────────────────────────────

    #[tokio::test]
    async fn test_mempalace_kg_add() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let args = json!({ "subject": "Rust", "predicate": "is_a", "object": "language" });
        let res = server.mempalace_kg_add(&args).await.unwrap();
        assert_eq!(res["status"], "success");
        assert!(res["triple_id"].is_string());
    }

    #[tokio::test]
    async fn test_mempalace_kg_query() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        // Add then query
        server
            .mempalace_kg_add(&json!({
                "subject": "Rust", "predicate": "is_a", "object": "language"
            }))
            .await
            .unwrap();

        let res = server
            .mempalace_kg_query(&json!({ "entity": "Rust" }))
            .await
            .unwrap();
        let results = res["results"].as_array().unwrap();
        assert!(!results.is_empty());
        assert_eq!(results[0]["subject"], "Rust");
    }

    #[tokio::test]
    async fn test_mempalace_kg_query_direction_filter() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        server
            .mempalace_kg_add(&json!({
                "subject": "A", "predicate": "knows", "object": "B"
            }))
            .await
            .unwrap();

        let outgoing = server
            .mempalace_kg_query(&json!({ "entity": "A", "direction": "outgoing" }))
            .await
            .unwrap();
        assert!(!outgoing["results"].as_array().unwrap().is_empty());

        let incoming = server
            .mempalace_kg_query(&json!({ "entity": "A", "direction": "incoming" }))
            .await
            .unwrap();
        // A has no incoming edges
        assert!(incoming["results"].as_array().unwrap().is_empty());
    }

    #[tokio::test]
    async fn test_mempalace_kg_invalidate() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        server
            .mempalace_kg_add(&json!({
                "subject": "X", "predicate": "is", "object": "Y"
            }))
            .await
            .unwrap();

        let res = server
            .mempalace_kg_invalidate(&json!({
                "subject": "X", "predicate": "is", "object": "Y"
            }))
            .await
            .unwrap();
        assert_eq!(res["status"], "success");
    }

    #[tokio::test]
    async fn test_mempalace_kg_invalidate_missing_fields() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        assert!(server
            .mempalace_kg_invalidate(&json!({"subject": "X"}))
            .await
            .is_err());
    }

    #[tokio::test]
    async fn test_mempalace_kg_timeline() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        server
            .mempalace_kg_add(&json!({
                "subject": "T", "predicate": "created_at", "object": "2024"
            }))
            .await
            .unwrap();

        let res = server
            .mempalace_kg_timeline(&json!({ "entity": "T" }))
            .await
            .unwrap();
        assert_eq!(res["entity"], "T");
        assert!(res["timeline"].is_array());
    }

    #[tokio::test]
    async fn test_mempalace_kg_stats() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_kg_stats().await.unwrap();
        assert!(res["entities"].is_number());
        assert!(res["triples"].is_number());
    }

    #[tokio::test]
    async fn test_mempalace_kg_full_lifecycle() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);

        // 1. Stats should be empty
        let stats = server.mempalace_kg_stats().await.unwrap();
        assert_eq!(stats["triples"], 0);

        // 2. Add triple
        let add = server
            .mempalace_kg_add(&json!({
                "subject": "mempalace", "predicate": "written_in", "object": "Rust"
            }))
            .await
            .unwrap();
        assert_eq!(add["status"], "success");

        // 3. Query it back
        let query = server
            .mempalace_kg_query(&json!({ "entity": "mempalace" }))
            .await
            .unwrap();
        let results = query["results"].as_array().unwrap();
        assert_eq!(results.len(), 1);
        assert_eq!(results[0]["object"], "Rust");

        // 4. Stats should reflect the addition
        let stats2 = server.mempalace_kg_stats().await.unwrap();
        assert_eq!(stats2["triples"], 1);

        // 5. Invalidate
        server
            .mempalace_kg_invalidate(&json!({
                "subject": "mempalace", "predicate": "written_in", "object": "Rust"
            }))
            .await
            .unwrap();

        // 6. Timeline should still show the entry (invalidated, not deleted)
        let timeline = server
            .mempalace_kg_timeline(&json!({ "entity": "mempalace" }))
            .await
            .unwrap();
        assert!(timeline["timeline"].is_array());
    }

    // ── Diary tests ──────────────────────────────────────────────────

    #[tokio::test]
    async fn test_mempalace_diary_write_and_read() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);

        let write_args = json!({ "agent": "test-agent", "content": "test diary entry" });
        server.mempalace_diary_write(&write_args).await.unwrap();

        let read_args = json!({ "agent": "test-agent", "last_n": 1 });
        let res = server.mempalace_diary_read(&read_args).await.unwrap();
        let entries = res["entries"].as_array().unwrap();
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0]["content"], "test diary entry");
    }

    #[tokio::test]
    async fn test_cli_diary_write_is_visible_to_mcp_diary_read_on_shared_storage() {
        let (config, _td) = setup_test();
        let config_dir = config.config_dir.clone();

        tokio::task::spawn_blocking(move || {
            cli_handlers::run(Cli {
                config_dir: Some(config_dir.clone()),
                command: Commands::Init(InitCommand {
                    name: "Tester".to_string(),
                }),
            })
        })
        .await
        .unwrap()
        .unwrap();

        let config_dir = config.config_dir.clone();
        tokio::task::spawn_blocking(move || {
            cli_handlers::run(Cli {
                config_dir: Some(config_dir),
                command: Commands::Diary(DiaryCommand {
                    command: DiarySubcommands::Write(DiaryWriteCommand {
                        content: "shared memory".to_string(),
                        tags: Some("focus".to_string()),
                        emotion: None,
                        wing: None,
                        room: None,
                    }),
                }),
            })
        })
        .await
        .unwrap()
        .unwrap();

        let config_dir = config.config_dir.clone();
        let server = McpServer::new_test(MempalaceConfig::new(Some(config_dir.clone())));
        let read = server
            .mempalace_diary_read(&json!({ "agent": "Tester", "last_n": 5 }))
            .await
            .unwrap();

        let entries = read["entries"].as_array().unwrap();
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0]["content"], "shared memory");

        let conn = rusqlite::Connection::open(config_dir.join("vectors.db")).unwrap();
        let count: i64 = conn
            .query_row(
                "SELECT COUNT(*) FROM memories WHERE source_file = 'diary://Tester'",
                [],
                |row| row.get(0),
            )
            .unwrap();
        assert_eq!(count, 1);
    }

    #[tokio::test]
    async fn test_mempalace_diary_write_with_metadata_fields() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);

        let write_args = json!({
            "agent": "meta-agent",
            "content": "test diary entry with metadata",
            "tags": ["focus", "journal"],
            "emotion": "joy",
            "timestamp": "2026-04-14T08:30:00Z",
            "room": "journal"
        });
        let write_res = server.mempalace_diary_write(&write_args).await.unwrap();
        assert_eq!(write_res["status"], "success");
        assert!(write_res["memory_id"].as_i64().unwrap() > 0);

        let read_args = json!({ "agent": "meta-agent", "last_n": 1 });
        let res = server.mempalace_diary_read(&read_args).await.unwrap();
        let entries = res["entries"].as_array().unwrap();
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0]["content"], "test diary entry with metadata");
        assert_eq!(entries[0]["emotion"], "joy");
        assert_eq!(entries[0]["emotion_code"], "joy");
        assert_eq!(entries[0]["room"], "journal");
    }

    #[tokio::test]
    async fn test_mempalace_diary_multiple_entries() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);

        for i in 0..5 {
            server
                .mempalace_diary_write(&json!({
                    "agent": "multi-agent",
                    "content": format!("entry {}", i)
                }))
                .await
                .unwrap();
        }

        let res = server
            .mempalace_diary_read(&json!({ "agent": "multi-agent", "last_n": 3 }))
            .await
            .unwrap();
        let entries = res["entries"].as_array().unwrap();
        assert_eq!(entries.len(), 3);
    }

    #[tokio::test]
    async fn test_mempalace_diary_read_empty() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server
            .mempalace_diary_read(&json!({ "agent": "ghost-agent" }))
            .await
            .unwrap();
        assert_eq!(res["entries"].as_array().unwrap().len(), 0);
    }

    #[tokio::test]
    async fn test_mempalace_diary_default_last_n() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        // Should default to 5 when last_n not provided
        let res = server
            .mempalace_diary_read(&json!({ "agent": "default-agent" }))
            .await
            .unwrap();
        assert!(res["entries"].is_array());
    }

    // ── Prune test ───────────────────────────────────────────────────

    #[tokio::test]
    async fn test_mempalace_prune_dry_run() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let args = json!({ "threshold": 0.9, "dry_run": true });
        let res = server.mempalace_prune(&args).await.unwrap();
        assert_eq!(res["status"], "success");
        assert_eq!(res["dry_run"], true);
    }

    #[tokio::test]
    async fn test_mempalace_prune_defaults() {
        let (config, _td) = setup_test();
        let server = McpServer::new_test(config);
        let res = server.mempalace_prune(&json!({})).await.unwrap();
        assert_eq!(res["dry_run"], true); // default is dry_run=true
    }

    // ── Content wrapper via tools/call for each tool ─────────────────

    #[tokio::test]
    async fn test_content_wrapper_all_parameterless_tools() {
        let (config, _td) = setup_test();
        let mut server = McpServer::new_test(config);

        let parameterless_tools = [
            "mempalace_status",
            "mempalace_list_wings",
            "mempalace_get_taxonomy",
            "mempalace_find_tunnels",
            "mempalace_graph_stats",
            "mempalace_get_aaak_spec",
            "mempalace_kg_stats",
        ];

        for tool_name in &parameterless_tools {
            let req = make_request(
                "tools/call",
                Some(json!({ "name": tool_name, "arguments": {} })),
                Some(json!(tool_name.to_string())),
            );
            let resp = server.handle_request(req).await;
            assert!(
                resp.error.is_none(),
                "tool {} returned error: {:?}",
                tool_name,
                resp.error
            );
            let res = resp.result.unwrap();
            let content = res["content"]
                .as_array()
                .unwrap_or_else(|| panic!("tool {} missing content array", tool_name));
            assert_eq!(
                content[0]["type"], "text",
                "tool {} content type wrong",
                tool_name
            );
            let text = content[0]["text"].as_str().unwrap();
            let _parsed: Value = serde_json::from_str(text)
                .unwrap_or_else(|_| panic!("tool {} text not valid JSON", tool_name));
        }
    }
}
