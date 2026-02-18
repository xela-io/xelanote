// Shared API types for xelanote

export interface User {
  id: number;
  username: string;
  email: string;
  is_admin: boolean;
  encryption_salt?: string; // Base64-encoded salt for E2E encryption (optional)
}

export interface AuthResponse {
  access_token?: string;
  refresh_token?: string;
  user?: User;
  requires_two_factor?: boolean;
  two_factor_methods?: string[];
  pending_login_token?: string;
  encryption_salt?: string; // Base64-encoded salt for E2E encryption
}

// SEC-001: Fields are optional — web clients receive empty JSON (tokens only in cookies)
export interface RefreshResponse {
  access_token?: string;
  refresh_token?: string;
}

export type RefreshResult =
  | { success: true; tokens: RefreshResponse }
  | { success: false; reason: 'auth_error' | 'network_error' | 'server_error' | 'timeout' };

export interface TwoFactorSetup {
  secret: string;
  qr_code_url: string;
  backup_codes: string[];
}

export interface TwoFactorStatus {
  enabled: boolean;
  totp_enabled: boolean;
  fido2_enabled: boolean;
  fido2_key_count: number;
  verified_at: string;
  unused_backup_codes: number;
}

export interface FIDO2CredentialInfo {
  id: number;
  device_name: string;
  created_at: string;
  last_used_at?: string;
  transports?: string[];
}

export interface Note {
  id: string;
  title: string;
  content: string;
  folder_path: string;
  display_order?: number;
  color?: string | null;
  version: number;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
  // Encryption fields
  encrypted_content?: string; // Base64
  content_encrypted?: boolean;
  encrypted_title?: string | null; // JSON string
  title_encrypted?: boolean;
  wrapped_dek?: string; // Base64
  encryption_version?: number;
  encryption_metadata?: string; // JSON
  // Summary fields (LLM-generated)
  summary?: string | null;
  encrypted_summary?: string | null;
  summary_encrypted?: boolean;
  content_hash?: string | null;
  summary_generated_at?: string | null;
  // Journal fields
  note_type?: string; // "note" (default) or "journal"
  journal_date?: string; // YYYY-MM-DD for journal notes
  // AI-Enabled (Claude API opt-in)
  ai_enabled?: boolean; // true = Cloud-KI (Claude) allowed for this note
  // Delta-sync field
  is_deleted?: boolean; // true for soft-deleted notes in delta-sync responses
  // Sharing fields
  is_shared?: boolean; // true if the note is shared with current user
  share_role?: 'viewer' | 'editor'; // recipient role on shared notes
}

export interface Backlink {
  id: string;
  title: string;
}

export interface SearchResult {
  id: string;
  title: string;
  snippet: string;
  rank: number;
  encrypted?: boolean;
  title_encrypted?: boolean;
  encrypted_title?: string | null;
  matched_keywords?: string[];
}

export interface RenameResult {
  note: Note;
  updated_note_count: number;
}

export interface Job {
  id: string;
  type: string;
  user_id: number;
  status: 'pending' | 'running' | 'completed' | 'failed';
  progress: number;
  result?: unknown;
  error?: string;
  created_at: string;
  updated_at: string;
  metadata: Record<string, unknown>;
}

export interface FolderInfo {
  path: string;
  note_count: number;
}

export interface Folder {
  id: number;
  path: string;
  parent_id?: number;
  name: string;
  note_count: number;
  display_order?: number;
  color?: string | null;
  created_at: string;
  updated_at: string;
  // AI-Enabled Default (Claude API opt-in)
  ai_enabled_default?: boolean; // New notes in this folder inherit this setting
  // Encryption Default
  encryption_default?: boolean; // New notes in this folder inherit this setting (true=encrypted)
}

export interface GraphNode {
  id: string;
  title: string;
  folder_path: string;
  is_resolved: boolean;
}

export interface GraphEdge {
  source_id: string;
  target_id: string;
  type: 'resolved' | 'unresolved';
}

export interface GraphMetadata {
  node_count: number;
  edge_count: number;
  truncated: boolean;
}

export interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
  metadata: GraphMetadata;
}

export interface Tag {
  id: number;
  name: string;
  user_id: number;
}

export interface Template {
  id: number;
  user_id: number;
  name: string;
  description: string;
  title: string;
  content: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTemplateRequest {
  name: string;
  description: string;
  title: string;
  content: string;
}

export interface UpdateTemplateRequest {
  name: string;
  description: string;
  title: string;
  content: string;
}

export interface Snippet {
  id: number;
  user_id: number;
  name: string;
  description: string;
  content: string;
  shortcut: string;
  created_at: string;
  updated_at: string;
}

export interface CreateSnippetRequest {
  name: string;
  description: string;
  content: string;
  shortcut?: string;
}

export interface UpdateSnippetRequest {
  name: string;
  description: string;
  content: string;
  shortcut?: string;
}

export interface QuickSearchFilters {
  query?: string;
  folders?: string[];
  tags?: string[];
  created_after?: string;
  created_before?: string;
  updated_after?: string;
  updated_before?: string;
}

export interface NotePayload {
  title: string;
  content?: string; // Optional for encrypted notes
  folder_path?: string;
  // Encryption fields
  encrypted_title?: string | null;
  title_encrypted?: boolean;
  encrypted_content?: string; // Base64
  wrapped_dek?: string; // Base64
  encryption_metadata?: string; // JSON
  keywords?: string[];
  // Client-side extracted links (for E2E encrypted notes where server can't parse content)
  links?: Array<{ target_title: string }>;
  // Client-side extracted due dates (for E2E encrypted notes)
  due_dates?: Array<{
    due_date: string;
    line_text: string;
    line_index: number;
    is_task_item: boolean;
    is_completed: boolean;
  }>;
  // Journal fields
  note_type?: string; // "note" (default) or "journal"
  journal_date?: string; // YYYY-MM-DD for journal notes
}

export interface AppConfig {
  captcha_enabled: boolean;
  captcha_site_key?: string;
  captcha_iframe_url?: string;
  version?: string;
  error_reporting_enabled?: boolean;
}

export interface UploadResponse {
  url: string;
  filename: string;
}

export interface ImportFile {
  path: string;
  filename: string;
  content: string;
}

export interface ImportResult {
  imported: number;
  skipped: number;
  failed: number;
  folders_created: number;
  errors?: string[];
}

export interface DueDateItem {
  id: number;
  note_id: string;
  note_title: string;
  due_date: string;
  line_text: string;
  line_index: number;
  is_task_item: boolean;
  is_completed: boolean;
}

export interface NoteVersion {
  id: number;
  note_id: string;
  user_id: number;
  version: number;
  title: string;
  content: string;
  snapshot_at: string;
  // Encryption fields (only present for encrypted notes)
  encrypted_content?: string;
  wrapped_dek?: string;
  content_encrypted?: boolean;
  title_encrypted?: boolean;
  encrypted_title?: string | null;
  encryption_version?: number;
}

export interface VersionListResponse {
  versions: NoteVersion[];
  next_cursor?: string;
  total: number;
}

export interface CompareResponse {
  version1: NoteVersion;
  version2: NoteVersion;
}

export interface WebAuthnCredentialInfo {
  id: number;
  credential_id: string;
  device_name: string;
  created_at: string;
  last_used_at?: string; // CRITICAL: Display in Settings UI for device auditing
}

export interface UserPreferences {
  theme: string;
  editor_mode: 'edit' | 'preview' | 'split' | 'live';
  keywords_enabled: boolean;
  encrypt_titles: boolean;
  security_level: 'paranoid' | 'balanced' | 'convenient';
  auto_lock_timeout: number; // minutes (0 = never)
  webauthn_credentials: WebAuthnCredentialInfo[];
  created: boolean;
}

export interface UpdatePreferencesRequest {
  theme: string;
  editor_mode: 'edit' | 'preview' | 'split' | 'live';
}

export interface UpdateSecurityPreferencesRequest {
  security_level?: string;
  auto_lock_timeout?: number;
}

export interface UpdateEncryptionPreferencesRequest {
  keywords_enabled: boolean;
  encrypt_titles: boolean;
}

export interface ClaudeAPIKeyStatus {
  has_key: boolean;
  updated_at?: string;
  masked_key?: string; // e.g., "sk-ant-api0...xxxx"
}

export interface GeminiAPIKeyStatus {
  has_key: boolean;
  updated_at?: string;
  masked_key?: string; // e.g., "AIzaSy...xxxx"
}

export interface AdminStats {
  total_users: number;
  total_notes: number;
  total_folders: number;
  total_tags: number;
  storage_used_mb: number;
}

export interface DailyCount {
  date: string;
  count: number;
}

export interface DailyFloat {
  date: string;
  value: number;
}

export interface DetailedStats {
  stats: AdminStats;
  user_growth: DailyCount[];
  note_growth: DailyCount[];
  storage_trend: DailyFloat[];
}

export interface AdminUser {
  id: number;
  username: string;
  email: string;
  is_admin: boolean;
  note_count: number;
  storage_mb: number;
  created_at: string;
  totp_enabled: boolean;
  totp_verified_at?: string;
}

export interface ActivityLog {
  id: number;
  user_id: number | null;
  username: string | null;
  action: string;
  target_type: string | null;
  target_id: string | null;
  details: Record<string, unknown> | null;
  ip_address: string | null;
  user_agent: string | null;
  created_at: string;
}

export interface ActivityLogsResponse {
  logs: ActivityLog[];
  total: number;
}

export interface SystemSettings {
  registration_enabled: string;
  max_notes_per_user: string;
  max_storage_mb_per_user: string;
  maintenance_mode: string;
  activity_retention_days: string;
}

export interface ActivityLogsOptions {
  limit?: number;
  page?: number;
  action?: string;
  user_id?: number;
  target_type?: string;
  date_from?: string;
  date_to?: string;
}

export interface SummarizeRequest {
  // For encrypted notes: decrypted content from frontend
  plaintext_content?: string;
  // For E2E notes: hash of plaintext (computed by frontend before encryption)
  plaintext_content_hash?: string;
  // For E2E notes: already encrypted summary (encrypted by frontend)
  encrypted_summary?: string;
}

export interface SummarizeResponse {
  summary: string;
}

export interface TagSuggestion {
  name: string;
  is_new: boolean;
  score: number;
}

export interface SuggestTagsResponse {
  suggestions: TagSuggestion[];
}

export interface LinkSuggestion {
  term: string;
  target_title: string;
  confidence: number;
}

export interface SuggestLinksResponse {
  suggestions: LinkSuggestion[];
}

export interface SpellIssue {
  byte_offset: number;
  byte_length: number;
  original: string;
  message: string;
  suggestions: string[];
  type: 'spelling' | 'grammar';
}

export interface SpellCheckResponse {
  issues: SpellIssue[];
}

export interface NoteTitleInfo {
  id: string;
  title: string;
  encrypted: boolean;
}

export interface UserFeature {
  user_id: number;
  feature: string;
  enabled: boolean;
  settings?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface JournalLookupResponse {
  exists: boolean;
  date: string;
  note_id: string; // Empty if not exists
}

export interface JournalCalendarResponse {
  year: number;
  month: number;
  dates: string[];
}

export interface JournalYearCalendarResponse {
  year: number;
  dates: string[];
}

export interface JournalEntry {
  id: string;
  title: string;
  journal_date: string;
  note_type: string;
  folder_path: string;
  created_at: string;
  updated_at: string;
  content_encrypted: boolean;
}

export interface JournalEntriesResponse {
  entries: JournalEntry[];
}

export interface AIEnabledResponse {
  ai_enabled: boolean;
}

export interface AIEnabledUpdateResponse {
  status: string;
  ai_enabled: boolean;
}

export interface FormatMarkdownResponse {
  formatted_content: string;
}

export type AIAction =
  | 'format'
  | 'summarize'
  | 'expand'
  | 'translate_de'
  | 'translate_en'
  | 'formal'
  | 'informal'
  | 'custom';

export interface AITransformResponse {
  transformed_content: string;
}

export interface TaskEventPayload {
  task_text?: string;
  task_index: number;
  encrypted_task_text?: string;
  wrapped_dek?: string;
  encryption_metadata?: string;
  event_type: 'completed' | 'reopened';
}

export interface NoteShare {
  id: number;
  note_id: string;
  owner_user_id: number;
  owner_username: string;
  shared_with_user_id: number;
  shared_with_username: string;
  role: 'viewer' | 'editor';
  created_at: string;
}

export interface SharedNote {
  id: string;
  title: string;
  content: string;
  folder_path: string;
  version: number;
  created_at: string;
  updated_at: string;
  note_type?: string;
  shared_by: string;
  share_role: 'viewer' | 'editor';
  share_id: number;
}

export interface UserSearchResult {
  id: number;
  username: string;
}

export interface FolderShare {
  id: number;
  folder_id: number;
  folder_path: string;
  folder_name: string;
  owner_user_id: number;
  owner_username: string;
  shared_with_user_id: number;
  shared_with_username: string;
  role: 'viewer' | 'editor';
  created_at: string;
  updated_at: string;
}

export interface SharedFolder {
  id: number;
  path: string;
  name: string;
  note_count: number;
  shared_by: string;
  share_role: 'viewer' | 'editor';
  share_id: number;
  created_at: string;
  updated_at: string;
}

export interface RecipeMetadata {
  note_id: string;
  user_id: number;
  servings: number;
  prep_time_minutes?: number | null;
  cook_time_minutes?: number | null;
  source_url?: string | null;
  difficulty?: 'easy' | 'medium' | 'hard' | null;
  updated_at: string;
}

export interface RecipeIngredient {
  id?: number;
  note_id?: string;
  amount?: number | null;
  amount_text?: string | null;
  unit?: string | null;
  name: string;
  group_name?: string | null;
  display_order: number;
  optional: boolean;
  scalable: boolean;
}

export interface ScaledIngredient extends RecipeIngredient {
  scaled_amount?: number | null;
  display_amount: string;
}

export interface RecipeCollection {
  id: number;
  user_id: number;
  name: string;
  description?: string | null;
  color?: string | null;
  display_order: number;
  recipe_count?: number;
}

export interface RecipeImage {
  id: number;
  note_id: string;
  user_id: number;
  image_url: string;
  caption?: string | null;
  display_order: number;
  created_at: string;
}

export interface RecipeDetail {
  note: Note;
  metadata: RecipeMetadata | null;
  ingredients: RecipeIngredient[];
  images: RecipeImage[];
  collections: RecipeCollection[];
  encrypted: boolean;
}

export interface RecipeListItem {
  id: string;
  title: string;
  folder_path: string;
  note_type: string;
  servings: number;
  prep_time_minutes?: number | null;
  cook_time_minutes?: number | null;
  difficulty?: string | null;
  updated_at: string;
  content_encrypted: boolean;
}

export interface CollectionShare {
  id: number;
  collection_id: number;
  collection_name: string;
  owner_user_id: number;
  owner_username: string;
  shared_with_user_id: number;
  shared_with_username: string;
  role: 'viewer' | 'editor';
  created_at: string;
  updated_at: string;
}

export interface SharedCollection {
  id: number;
  name: string;
  description?: string | null;
  color?: string | null;
  recipe_count: number;
  shared_by: string;
  share_role: 'viewer' | 'editor';
  share_id: number;
  created_at: string;
  updated_at: string;
}

export interface SimilarRecipeResult {
  note_id: string;
  title: string;
  similarity_score: number;
  reason: string;
}

export interface IngredientMatchResult {
  note_id: string;
  title: string;
  match_score: number;
  matched_ingredients: string[];
  missing_ingredients: string[];
}

export interface GeneratedIngredient {
  name: string;
  amount?: number | null;
  unit?: string | null;
  group_name?: string | null;
  scalable: boolean;
  optional: boolean;
}

export interface GeneratedRecipe {
  title: string;
  servings: number;
  prep_time_minutes?: number | null;
  cook_time_minutes?: number | null;
  difficulty?: 'easy' | 'medium' | 'hard' | null;
  source_url?: string | null;
  ingredients: GeneratedIngredient[];
  instructions: string;
}

export interface IngredientSuggestionResult {
  matches: IngredientMatchResult[];
  generated: GeneratedRecipe[];
}

// === JSON Canvas spec 1.0 types (https://jsoncanvas.org/spec/1.0/) ===

export type CanvasNodeType = 'text' | 'file' | 'link' | 'group';
export type CanvasSide = 'top' | 'right' | 'bottom' | 'left';
export type CanvasEndpoint = 'none' | 'arrow';
export type CanvasColor = '1' | '2' | '3' | '4' | '5' | '6' | string;

export interface CanvasNodeBase {
  id: string;
  type: CanvasNodeType;
  x: number;
  y: number;
  width: number;
  height: number;
  color?: CanvasColor;
}

export interface CanvasTextNode extends CanvasNodeBase {
  type: 'text';
  text: string;
}

export interface CanvasFileNode extends CanvasNodeBase {
  type: 'file';
  file: string;
  subpath?: string;
  'x-xelanote-note-id'?: string;
}

export interface CanvasLinkNode extends CanvasNodeBase {
  type: 'link';
  url: string;
}

export interface CanvasGroupNode extends CanvasNodeBase {
  type: 'group';
  label?: string;
  background?: string;
  backgroundStyle?: 'cover' | 'ratio' | 'repeat';
}

export type CanvasNode = CanvasTextNode | CanvasFileNode | CanvasLinkNode | CanvasGroupNode;

export interface CanvasEdge {
  id: string;
  fromNode: string;
  toNode: string;
  fromSide?: CanvasSide;
  toSide?: CanvasSide;
  fromEnd?: CanvasEndpoint;
  toEnd?: CanvasEndpoint;
  color?: CanvasColor;
  label?: string;
}

export interface CanvasData {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
}
