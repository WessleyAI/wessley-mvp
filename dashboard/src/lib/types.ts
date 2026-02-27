// metrics-latest.json
export interface TopMake {
  name: string;
  models: number;
  documents: number;
}
export interface TopVehicle {
  vehicle: string;
  documents: number;
  components: number;
}
export interface KnowledgeGraph {
  total_nodes: number;
  total_relationships: number;
  nodes_by_type: Record<string, number>;
  relationships_by_type: Record<string, number>;
  top_makes: TopMake[];
  top_vehicles: TopVehicle[];
  recent_vehicles: string[];
}
export interface VectorStore {
  total_vectors: number;
  collection: string;
  dimensions: number;
  status: string;
}
export interface Ingestion {
  total_docs_ingested: number;
  total_errors: number;
  docs_by_source: Record<string, number>;
  last_ingestion: string;
  files_processed: number;
}
export interface ScraperStatus {
  status: string;
  last_scrape: string;
  total_posts?: number;
  total_docs?: number;
}
export interface ManualsScraper {
  discovered: number;
  downloaded: number;
  ingested: number;
  failed: number;
}
export interface Scrapers {
  reddit: ScraperStatus;
  nhtsa: ScraperStatus;
  ifixit: ScraperStatus;
  youtube: ScraperStatus;
  manuals: ManualsScraper;
}
export interface ServiceStatus {
  status: string;
  version?: string;
  vectors?: number;
  model?: string;
}
export interface Infrastructure {
  neo4j: ServiceStatus;
  qdrant: ServiceStatus;
  ollama: ServiceStatus;
}
export interface Metrics {
  timestamp: string;
  knowledge_graph: KnowledgeGraph;
  vector_store: VectorStore;
  ingestion: Ingestion;
  scrapers: Scrapers;
  infrastructure: Infrastructure;
}

// metrics-history.json
export interface MetricsHistoryEntry {
  timestamp: string;
  period: string;
  new_docs: number;
  new_nodes: number;
  new_relations: number;
  new_vectors: number;
  errors_delta: number;
  docs_by_source: Record<string, number>;
  new_vehicles?: string[];
  total_docs: number;
  total_vectors: number;
  total_nodes: number;
}

// analysis-latest.json
export interface CriticalItem {
  title: string;
  detail: string;
  fix: string;
}
export interface WarningItem {
  title: string;
  detail: string;
}
export interface Suggestion {
  title: string;
  impact: string;
  effort: string;
}
export interface BugItem {
  title: string;
  file: string;
  line: number;
  detail: string;
  fix: string;
}
export interface AnalysisMetrics {
  total_docs: number;
  total_nodes: number;
  total_vectors: number;
  total_relationships: number;
  error_rate: number;
  sources_active: number;
  sources_dead: number;
  embedding_ratio: number;
  nhtsa_pct: number;
  docs_last_5min: number;
  vectors_last_5min: number;
}
export interface Analysis {
  timestamp: string;
  critical: CriticalItem[];
  warnings: WarningItem[];
  healthy: string[];
  suggestions: Suggestion[];
  bugs: BugItem[];
  metrics: AnalysisMetrics;
  strategy: string[];
}

// fixes-history.json
export interface FixedItem {
  title: string;
  category: string;
  detail: string;
  files_changed: string[];
}
export interface SkippedItem {
  title: string;
  reason: string;
}
export interface FixRun {
  timestamp: string;
  analysis_timestamp: string;
  fixed: FixedItem[];
  skipped: SkippedItem[];
  tests_before: { passed: number; failed: number };
  tests_after: { passed: number; failed: number };
  build_status: string;
  commit: string;
  summary: string;
}

// logs-latest.json
export interface LogEntry {
  time: string;
  category: string;
  message: string;
}

export interface DashboardData {
  metrics: Metrics | null;
  history: MetricsHistoryEntry[];
  analysis: Analysis | null;
  fixes: FixRun[];
  logs: LogEntry[];
  loading: boolean;
  error: string | null;
  lastRefresh: number;
}
