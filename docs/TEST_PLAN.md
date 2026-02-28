# Wessley MVP — Comprehensive Test Plan

> **Purpose:** Detailed specification of every test needed for MVP coverage. Each entry is implementation-ready — no guessing required.
>
> **Date:** 2025-02-24 | **Branch:** `docs/test-plan`

---

## Table of Contents

1. [Unit Tests](#1-unit-tests)
2. [Integration Tests](#2-integration-tests)
3. [Test Infrastructure](#3-test-infrastructure)
4. [Priority Matrix](#4-priority-matrix)

---

## 1. Unit Tests

### 1.1 `pkg/fn` — Functional Utilities

**Already tested (18 existing test files covering):**
- Result: Ok/Err/Errf/Must/UnwrapOr/Map/AndThen/MapResult/FromPair/Collect
- Slice: Map/Filter/FilterMap/Reduce/GroupBy/Chunk/Unique/UniqueBy/FlatMap
- Parallel: ParMap/ParMapEmpty/ParMapUnbounded/ParMapResult/FanOut/FanOutResult
- Pipeline: Then/ThenShortCircuits/Pipeline/MapStage/TapStage/BatchStage/TracedStage
- Retry: RetrySuccess/RetryExhausted/RetryContextCancelled/RetryStage

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestParMap_PanicInWorker` | ParMap doesn't deadlock if worker panics | Pass func that panics for one item | Should panic (or recover gracefully — document behavior) |
| `TestParMap_SingleItem` | Edge case with 1 item | `ParMap([]int{5}, 2, double)` | Returns `[10]` |
| `TestParMapResult_MixedErrors` | Error handling in parallel results | Mix Ok/Err results | Each Result independent, no short-circuit |
| `TestFanOut_Empty` | Zero functions | `FanOut[int]()` | Returns empty slice |
| `TestCollect_SingleError` | Single-element error slice | `Collect([]Result{Err(...)})` | Returns Err |
| `TestPipeline_Empty` | Zero stages | `Pipeline[int]()` on input 5 | Returns Ok(5) |
| `TestPipeline_SingleStage` | One stage | Single increment | Returns Ok(input+1) |
| `TestRetry_ImmediateSuccess` | No retries needed | func succeeds first try | Returns Ok, 1 attempt |
| `TestRetry_MaxWaitCap` | Backoff doesn't exceed MaxWait | Set MaxWait=10ms, InitialWait=5ms, 5 attempts | Sleep never exceeds 10ms |
| `TestBatchStage_Empty` | Empty input slice | `BatchStage(2, stage)(ctx, []int{})` | Returns Ok([]) |
| `TestBatchStage_ErrorPropagation` | One item errors | Stage errors on item 2 of 3 | Returns first Err from Collect |
| `TestChunk_ExactMultiple` | Slice length divisible by n | `Chunk([1,2,3,4], 2)` | `[[1,2], [3,4]]` |
| `TestChunk_SingleElement` | Single-element slice | `Chunk([1], 5)` | `[[1]]` |
| `TestFilter_AllMatch` | Every element passes | `Filter([1,2,3], func always true)` | Returns all |
| `TestFilter_NoneMatch` | No elements pass | `Filter([1,2,3], func always false)` | Returns nil/empty |
| `TestReduce_Empty` | Empty slice | `Reduce([]int{}, 0, add)` | Returns init value 0 |
| `TestGroupBy_Empty` | Empty slice | `GroupBy([]int{}, key)` | Returns empty map |
| `TestFlatMap_Empty` | Empty slice | `FlatMap([]int{}, f)` | Returns nil |
| `TestFlatMap_EmptyInner` | Func returns empty slices | `FlatMap([1,2], func returns [])` | Returns nil |
| `TestMapResult_Error` | Error propagation | `MapResult(Err(...), f)` | Returns Err, f not called |

---

### 1.2 `pkg/mid` — HTTP Middleware

**Already tested:**
- ChainOrder, LoggerCapturesStatus, RecoverCatchesPanic, CORSOptionsReturns204, CORSNonOptionsPassesThrough

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestChain_NoMiddleware` | Chain with zero middlewares | `Chain(handler)` | Handler called directly |
| `TestChain_SingleMiddleware` | Chain with one middleware | Single mw wrapping handler | mw runs then handler |
| `TestLogger_DefaultStatusOK` | Status defaults to 200 when handler doesn't call WriteHeader | Handler just writes body | Log shows status=200 |
| `TestLogger_500Error` | Logs 500 status | Handler writes 500 | Log captures 500 |
| `TestRecover_NoPanic` | Handler without panic | Normal handler | Passes through, returns original status |
| `TestRecover_PanicWithString` | Panic with string value | `panic("string error")` | Returns 500, logs error |
| `TestRecover_PanicWithError` | Panic with error value | `panic(errors.New("err"))` | Returns 500, logs error |
| `TestCORS_HeadersOnGet` | CORS headers set on GET | GET request, origin="https://app.com" | All CORS headers present |
| `TestCORS_AllowMethodsHeader` | Allow-Methods header correctness | OPTIONS request | Contains GET, POST, PUT, DELETE, OPTIONS, PATCH |
| `TestStatusWriter_DoubleWriteHeader` | Only first WriteHeader takes effect | Call WriteHeader(201) then WriteHeader(500) | Status captured as 201 |
| `TestStatusWriter_WriteWithoutHeader` | Write without explicit WriteHeader | Call Write(bytes) directly | Status defaults to 200 |
| `TestOTel_WrapsHandler` | OTel middleware creates spans | Handler with OTel middleware | No panic, handler still executes (span verification optional) |

---

### 1.3 `pkg/natsutil` — NATS Typed Helpers

**Already tested:**
- NatsHeaderCarrier (Set/Get/Keys), NilHeader, PublishSerializesJSON, SubscribeDropsMalformed

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestNatsHeaderCarrier_MultipleKeys` | Multiple headers | Set 3 different keys | Keys() returns all 3 |
| `TestNatsHeaderCarrier_OverwriteKey` | Set same key twice | Set "key" to "a" then "b" | Get returns "b" |
| `TestPublish_MarshalError` | Unmarshalable type | Pass channel type to Publish | Returns json marshal error |
| `TestSubscribe_ValidMessage` | Full handler invocation | Manually invoke subscription handler logic with valid JSON | Handler called with deserialized value |
| `TestRequest_MarshalError` | Request with unmarshalable type | Pass channel type | Returns error |
| `TestRequest_ResponseUnmarshalError` | Invalid response JSON | Simulate NATS response with bad JSON | Returns unmarshal error |

> **Note:** Full Publish/Subscribe/Request tests require a NATS connection. See Integration Tests (§2) for those.

---

### 1.4 `pkg/repo` — Generic Repository & Neo4j Implementation

**Already tested:**
- NewNeo4jRepoDefaults, DefaultIDKey, WithIDKey option

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestNeo4jRepo_Get_NotFound` | Not found error | Mock driver: Run returns empty result | Returns `"<label> not found"` error |
| `TestNeo4jRepo_Get_Success` | Happy path | Mock driver: Run returns one record | Returns entity from fromRecord |
| `TestNeo4jRepo_Get_DriverError` | Driver failure | Mock driver: Run returns error | Returns driver error |
| `TestNeo4jRepo_List_DefaultLimit` | Default limit of 100 | ListOpts{Limit: 0} | Cypher uses LIMIT 100 |
| `TestNeo4jRepo_List_CustomOffset` | Offset pagination | ListOpts{Offset: 10, Limit: 5} | Cypher uses SKIP 10 LIMIT 5 |
| `TestNeo4jRepo_List_Empty` | Empty results | Mock: no rows | Returns nil/empty slice |
| `TestNeo4jRepo_Create_Success` | Create node | Mock: returns created record | Returns entity |
| `TestNeo4jRepo_Create_Failure` | Create with no result | Mock: Run ok but no Next | Returns "failed to create" error |
| `TestNeo4jRepo_Update_NotFound` | Update missing entity | Mock: no result | Returns "<label> not found" error |
| `TestNeo4jRepo_Update_Success` | Update existing | Mock: returns updated record | Returns updated entity |
| `TestNeo4jRepo_Delete_Success` | Delete by ID | Mock: Run succeeds | Returns nil |
| `TestNeo4jRepo_Delete_Error` | Delete driver failure | Mock: Run returns error | Returns error |
| `TestNeo4jRepo_CypherInjectionSafe` | Label/idKey used safely | Verify generated Cypher with known inputs | No injection possible (label set at construction) |

> **Mocking strategy:** Create a `mockDriverWithContext` implementing `neo4j.DriverWithContext` that returns a `mockSessionWithContext`. Session's `Run` returns a `mockResultWithContext`. This is the standard approach for neo4j-go-driver v5 testing.

---

### 1.5 `pkg/resilience` — Circuit Breaker & Rate Limiter

**Already tested (comprehensive):**
- Breaker: StartsClosed, TripsAfterThreshold, ResetsOnSuccess, HalfOpen, HalfOpenFailure, BreakerStage
- Limiter: Allow, Refill, Call, Wait, WaitCancelled, LimiterStage

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestBreaker_DefaultOpts` | Zero-value opts get defaults | `NewBreaker(BreakerOpts{})` | FailThreshold=5, Timeout=30s, HalfOpenMax=1 |
| `TestBreaker_StateString` | State.String() for all states | All 3 states + invalid | "closed", "open", "half-open", "unknown" |
| `TestBreaker_ConcurrentCalls` | Thread safety | 100 goroutines calling simultaneously | No race conditions (use `-race`) |
| `TestBreaker_HalfOpenMaxExceeded` | Excess probes rejected in half-open | Set HalfOpenMax=1, make 2 calls in half-open | 2nd call returns ErrCircuitOpen |
| `TestBreaker_CallResult_Success` | Generic CallResult with fn.Result | Success function | Returns Ok result |
| `TestBreaker_CallResult_CircuitOpen` | CallResult when open | Tripped breaker | Returns Err(ErrCircuitOpen) |
| `TestBreaker_CallResult_TripsAndRecovers` | Full lifecycle via CallResult | Fail → open → wait → half-open → succeed → closed | State transitions correct |
| `TestLimiter_BurstDefault` | Burst defaults to 1 if <=0 | `NewLimiter(LimiterOpts{Burst: 0})` | Burst becomes 1 |
| `TestLimiter_RefillCap` | Tokens don't exceed burst | Wait long time, check tokens | Tokens capped at burst |
| `TestLimiter_ConcurrentAllow` | Thread safety | 50 goroutines calling Allow() | No race, total allows ≤ burst |
| `TestLimiter_CallWait_Success` | CallWait blocks and succeeds | Rate=1000, drain 1 token | Returns nil quickly |
| `TestLimiter_CallWait_ContextCancelled` | CallWait respects context | Slow rate, short timeout | Returns context.DeadlineExceeded |
| `TestLimiterStageWait_Success` | LimiterStageWait happy path | Fast rate limiter | Returns stage result |
| `TestLimiterStageWait_ContextCancelled` | LimiterStageWait timeout | Slow rate, cancelled ctx | Returns context error |
| `TestLimiter_ZeroRate` | Rate=0 means no refill | Drain tokens, advance time | No new tokens |

---

### 1.6 `engine/domain` — Validation & Domain Types

**Already tested (good coverage):**
- ValidateVehicle: valid, invalid make/model/year/VIN
- ValidateQuery: valid, too short, injection (3 patterns), profanity, invalid vehicle
- ValidationError: Unwrap, errors.As
- SymptomAndFixCategories

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestValidateVehicle_BoundaryYears` | Exact boundary values | Year=1980 (min) and Year=2027 (max) | Both valid |
| `TestValidateVehicle_BoundaryYearsMinus1` | Just outside boundaries | Year=1979 and Year=2028 | Both ErrYearOutOfRange |
| `TestValidateVehicle_EmptyVIN` | VIN is optional | Vehicle with VIN="" | Valid |
| `TestValidateVehicle_VINWithQ` | Forbidden Q in VIN | VIN with Q character | ErrInvalidVIN |
| `TestValidateVehicle_VINWithO` | Forbidden O in VIN | VIN with O character | ErrInvalidVIN |
| `TestValidateVehicle_VINLowercase` | Lowercase VIN accepted | Valid VIN in lowercase | Valid (uppercased internally) |
| `TestValidateVehicle_VINWrongLength` | 16 or 18 character VIN | VIN of wrong length | ErrInvalidVIN |
| `TestValidateVehicle_CaseInsensitiveModel` | Model matching ignores case | "camry" for Toyota | Valid |
| `TestValidateVehicle_AllMakes` | Every make in SupportedMakes | Iterate all makes, use first model | All valid |
| `TestValidateQuery_ExactMinLength` | Exactly 5 runes | "abcde" with valid vehicle | Valid |
| `TestValidateQuery_Unicode` | Unicode rune counting | 4 emoji (4 runes < 5) | ErrQueryTooShort |
| `TestValidateQuery_InjectionSQL_UNION_SELECT` | UNION SELECT pattern | `"SELECT * UNION SELECT FROM"` | ErrQueryInjection |
| `TestValidateQuery_InjectionNoSQL_gte` | NoSQL $gte operator | `{"$gte": 0}` | ErrQueryInjection |
| `TestValidateQuery_InjectionTemplateVariable` | Template injection | `${env.SECRET}` | ErrQueryInjection |
| `TestValidateQuery_ProfanityWithPunctuation` | Profanity trimmed of punctuation | `"this damn! thing"` | ErrQueryProfanity |
| `TestValidateQuery_ProfanitySubstringNotMatched` | "class" doesn't match "ass" | `"the class of this engine"` | Valid (word-boundary) |
| `TestValidateQuery_CleanInjectionLookalike` | Safe text with SQL keywords | `"My car dropped in performance"` | Valid (no two-keyword pattern) |
| `TestValidateScrapedPost_Valid` | Valid ScrapedPost | All fields filled | nil error |
| `TestValidateScrapedPost_EmptyContent` | Empty content | Content="" | Error |
| `TestValidateScrapedPost_UnknownSource` | Invalid source | Source="tiktok" | Error |
| `TestValidateScrapedPost_EmptySourceID` | Empty SourceID | SourceID="" | Error |
| `TestValidateScrapedPost_EmptyTitle` | Empty title | Title="" | Error |
| `TestValidateScrapedPost_AllValidSources` | Each valid source | "reddit", "youtube", "forum" | All valid |

---

### 1.7 `engine/graph` — Neo4j Knowledge Graph

**Already tested:**
- sanitizeRelType (7 cases), componentFromProps, componentToMap, NewGraphStore(nil)

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestSanitizeRelType_SpecialChars` | Strips all non-alphanumeric/underscore | `"has-wire/power.on"` | `"HASWIREON"` → or just alphanumeric |
| `TestSanitizeRelType_Numbers` | Numbers preserved | `"TYPE_123"` | `"TYPE_123"` |
| `TestComponentFromProps_MissingFields` | Nil/missing properties | `map[string]any{}` | Component with empty strings |
| `TestComponentFromProps_NonStringPropValues` | Non-string prop_ values ignored | `{"prop_count": 42}` | Properties map empty (int not string) |
| `TestComponentToMap_NilProperties` | Component with nil Properties | `Component{Properties: nil}` | Map without prop_ keys |
| `TestStrProp_Missing` | strProp with missing key | props without key | Returns "" |
| `TestStrProp_NonString` | strProp with non-string value | `{"id": 42}` | Returns "" |
| `TestGraphStore_SaveComponent` | SaveComponent cypher | Mock driver, verify cypher & params | MERGE query with correct props |
| `TestGraphStore_SaveEdge` | SaveEdge cypher | Mock driver | MATCH + MERGE with sanitized rel type |
| `TestGraphStore_Neighbors_DefaultDepth` | depth<=0 defaults to 1 | Mock driver, depth=0 | Cypher uses `[*1..1]` |
| `TestGraphStore_Neighbors_CustomDepth` | Custom depth | depth=3 | Cypher uses `[*1..3]` |
| `TestGraphStore_FindByVehicle` | Vehicle key formatting | year=2020, make="Toyota", model="Camry" | vehicleKey = "2020-Toyota-Camry" |
| `TestGraphStore_TracePath_NoPath` | No path between nodes | Mock: no result | Returns "no path" error |
| `TestGraphStore_SaveBatch_Transaction` | Batch in single tx | Mock driver with ExecuteWrite | All components and edges in one tx |
| `TestGraphStore_SaveBatch_Empty` | Empty batch | Empty slices | No errors |

> **Mocking strategy:** For graph tests, create a mock `neo4j.DriverWithContext` → mock `SessionWithContext` → mock `ResultWithContext`. Verify the Cypher strings and parameters passed to `Run`/`ExecuteWrite`.

---

### 1.8 `engine/semantic` — Qdrant Vector Store

**Already tested (integration, requires QDRANT_ADDR):**
- UpsertAndSearch, UpsertEmpty, EnsureCollectionIdempotent

**Missing unit tests (with mocked gRPC clients):**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestVectorStore_Upsert_PayloadTypes` | All payload value types | Mock PointsClient; payload with string, int, int64, float64, bool, other | Each type maps to correct pb.Value kind |
| `TestVectorStore_Upsert_Error` | gRPC error handling | Mock PointsClient returns error | Returns wrapped error |
| `TestVectorStore_Search_EmptyResults` | No matches | Mock returns empty result | Returns empty slice |
| `TestVectorStore_Search_ScoreAndContent` | Result field mapping | Mock returns 2 results with payload | SearchResult fields correctly populated |
| `TestVectorStore_SearchFiltered_SingleFilter` | Single filter | filter={"source": "reddit"} | Filter has 1 Must condition |
| `TestVectorStore_SearchFiltered_MultipleFilters` | Multiple filters | 3 filters | Filter has 3 Must conditions |
| `TestVectorStore_SearchFiltered_NoFilters` | Nil filters | filters=nil | No Filter on request |
| `TestVectorStore_DeleteByDocID` | Delete filter construction | Mock PointsClient | PointsSelector with doc_id filter |
| `TestVectorStore_EnsureCollection_AlreadyExists` | Collection exists | Mock ListCollections returns matching name | No Create call |
| `TestVectorStore_EnsureCollection_Creates` | Collection doesn't exist | Mock ListCollections returns empty | Create called with Cosine distance, correct dims |
| `TestVectorStore_DeleteCollection` | Delete call | Mock CollectionsClient | Delete called with collection name |
| `TestVectorStore_Close` | Connection cleanup | Store with mock conn | conn.Close() called |
| `TestFieldMatch` | fieldMatch helper | key="source", value="reddit" | Returns correct Condition proto |

> **Mocking strategy:** Create mock implementations of `pb.PointsClient` and `pb.CollectionsClient` (gRPC interfaces). Inject via struct fields.

---

### 1.9 `engine/ingest` — Ingestion Pipeline

**Already tested:**
- ValidateStage (valid, invalid source, empty content)
- ParseStage, ChunkDocStage
- splitSentences (4 cases), chunkSentences overlap
- PipelineComposition (Validate → Parse → Chunk)

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestValidateStage_EmptySourceID` | Missing source_id | Post with SourceID="" | Returns Err |
| `TestValidateStage_EmptyTitle` | Missing title | Post with Title="" | Returns Err |
| `TestParseStage_MetadataPreserved` | Metadata fields carried through | Post with vehicle, author, url | ParsedDoc.Metadata has all fields |
| `TestParseStage_DocID_Format` | Doc ID = source:source_id | Post with Source="youtube", SourceID="abc" | ID = "youtube:abc" |
| `TestChunkDoc_ShortContent` | Content shorter than chunk size | Single short sentence | Single chunk fallback |
| `TestChunkDoc_EmptySentences` | Content that produces 0 sentences | "" or whitespace-only content (after validation) | Single chunk fallback with content |
| `TestChunkSentences_NoOverlap` | overlap=0 | Many sentences, overlap=0 | Chunks don't share sentences |
| `TestChunkSentences_LargeOverlap` | overlap > chunkSize | overlap=1000, chunkSize=50 | Still makes forward progress |
| `TestChunkSentences_SingleSentence` | One sentence | Single long sentence | One chunk |
| `TestChunkSentences_NegativeOverlap` | overlap=-1 | Negative overlap | Treated as 0 |
| `TestChunkSentences_ZeroChunkSize` | chunkSize=0 | Zero chunk size | Defaults to 512 |
| `TestSplitSentences_Abbreviations` | Periods in abbreviations | "Dr. Smith fixed the car." | Ideally 1 sentence (known limitation to document) |
| `TestSplitSentences_MultipleNewlines` | Multiple newlines | "Line1\n\nLine2\n\n\nLine3" | 3 non-empty sentences |
| `TestSplitSentences_QuestionAndExclamation` | Mixed punctuation | "What? Oh! OK." | 3 sentences |
| `TestWordCount` | Word counting | "hello world foo" | Returns 3 |
| `TestWordCount_Empty` | Empty string | "" | Returns 0 |
| `TestNewEmbed_Success` | Embed stage calls gRPC correctly | Mock EmbedServiceClient returning 3 embeddings | EmbeddedDoc has correct embeddings per chunk |
| `TestNewEmbed_BatchSize` | Batching over EmbedBatchSize | 150 chunks (>100 batch size) | 2 gRPC calls: 100 + 50 |
| `TestNewEmbed_GrpcError` | gRPC failure | Mock returns error | Returns Err with "embed batch" prefix |
| `TestNewStore_Success` | Store saves to graph + vector | Mock GraphStore & VectorStore | SaveComponent called, Upsert called with correct records |
| `TestNewStore_GraphError` | Graph save fails | Mock GraphStore returns error | Returns Err with "graph save" prefix |
| `TestNewStore_VectorError` | Vector upsert fails | Mock VectorStore returns error | Returns Err with "vector upsert" prefix |
| `TestNewPipeline_EndToEnd` | Full pipeline with mocks | Mock Embedder, VectorStore, GraphStore | Post → doc_id string |
| `TestStartConsumer_Dedup` | Deduplication skips known docs | DeduplicateF returns true | Pipeline not executed, message acked |
| `TestStartConsumer_DLQ` | Dead letter after MaxRetries | Pipeline always fails, retry count ≥ MaxRetries | DLQ message published |
| `TestStartConsumer_RetryHeader` | Retry count incremented | Pipeline fails once | Re-published message has X-Retry-Count=1 |
| `TestLoggedTap_LogsEntryExit` | Logging side-effect | Capture slog output | "stage.enter" and "stage.exit" logged |
| `TestParsedDocFromPost` | Conversion correctness | ScrapedPost with all fields | ParsedDoc has correct ID, Source, Title, Vehicle, Sentences, Metadata |

---

### 1.10 `engine/rag` — RAG Orchestration

**Already tested (good coverage):**
- Query: Success, WithoutGraph, EmbedError, SearchError, GraphFailureGraceful
- extractKeywords, buildContextParts

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestQuery_ChatError` | Chat service fails | Mock chat returns error | Returns "rag: chat" error |
| `TestQuery_EmptySearchResults` | No search hits | Mock returns empty results | Still calls chat with empty context (or just graph) |
| `TestQuery_VehicleFilter` | Vehicle filter passed to search | vehicle="2020-Toyota-Camry" | Search called with filter["vehicle"] |
| `TestQuery_NoVehicleFilter` | Empty vehicle | vehicle="" | Search called with empty filter map |
| `TestQuery_SearchTimeout` | Search exceeds timeout | Mock search blocks, short SearchTimeout | Returns context deadline error |
| `TestQuery_GraphNoComponents` | Graph returns empty | Mock graph returns empty components | No graph context in chat parts |
| `TestQuery_SourceMapping` | Sources correctly mapped | 3 search results | Answer.Sources has 3 entries with correct fields |
| `TestQuery_CustomOptions` | Custom TopK, Temperature, Model | Non-default options | Passed to search (topK) and chat (temperature, model) |
| `TestEnrichWithGraph_NoKeywords` | Short/stop-word question | "is it?" → no keywords | Returns "" |
| `TestEnrichWithGraph_FormatsComponents` | Component formatting | 2 components, 1 edge | String contains component names and edge info |
| `TestExtractKeywords_Empty` | Empty string | "" | Returns nil |
| `TestExtractKeywords_AllStopWords` | Only stop words | "the is a an" | Returns nil |
| `TestExtractKeywords_PunctuationStripped` | Punctuation removed | "ECU? wiring! sensor." | Returns ["ecu", "wiring", "sensor"] |
| `TestExtractKeywords_ShortWordsFiltered` | Words ≤2 chars removed | "I am OK no" | Returns nil (all ≤3 or stop words) |
| `TestBuildContextParts_Empty` | No results, no graph | Empty results, "" graph | Returns empty slice |
| `TestBuildContextParts_FormatIncludesScore` | Score in format string | Result with score=0.95 | Part contains "0.950" |
| `TestDefaultOptions` | Default values correct | Call DefaultOptions() | TopK=5, Temperature=0.3, MaxTokens=1024, UseGraph=true, SearchTimeout=5s |

---

### 1.11 `engine/scraper` — YouTube & Transcript Scraping

**Already tested:**
- CleanTranscript (4 cases), extractMetadata, extractMetadata_NoMatch

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestCleanTranscript_AllEntities` | All HTML entities | `&lt;&gt;&amp;&quot;&#39;` | `<>&"'` |
| `TestCleanTranscript_AllBracketNoise` | All bracket patterns | `[Music][Applause][Laughter][Cheering][Inaudible]` | All removed |
| `TestCleanTranscript_EmptyInput` | Empty string | "" | Returns "" |
| `TestCleanTranscript_NoNoise` | Clean text passes through | "Normal clean text here" | Same text |
| `TestExtractMetadata_MultipleSymptoms` | Multiple symptom matches | Text with "won't start", "check engine", "dead battery" | All 3 in Symptoms |
| `TestExtractMetadata_AllFixes` | All fix keywords | Text containing all 10 fix keywords | All found |
| `TestExtractMetadata_VehiclePattern_Variants` | Different year/make/model | "2023 Toyota Corolla", "1999 Ford F-150" | Vehicle extracted correctly |
| `TestExtractMetadata_VehiclePattern_NoMatch` | No year-make-model pattern | "fixing car problems general" | Vehicle="" |
| `TestGetTranscript_Success` | Successful transcript fetch | httptest.Server returning valid XML | Returns cleaned transcript |
| `TestGetTranscript_FallbackToASR` | First URL fails, ASR succeeds | httptest.Server: 404 for first, XML for second | Returns transcript from ASR |
| `TestGetTranscript_AllFail` | Both URLs fail | httptest.Server: 404 for both | Returns Err "no transcript available" |
| `TestGetTranscript_InvalidXML` | Malformed XML response | httptest.Server returns bad XML | Falls through to next URL or error |
| `TestGetTranscript_ShortResponse` | Response < 50 bytes | httptest.Server returns tiny body | Skipped, tries next URL |
| `TestYouTubeScraper_NewWithDefaults` | Default channels | `NewYouTubeScraper("key", nil)` | Uses DefaultYouTubeChannels |
| `TestYouTubeScraper_ScrapeVideo_Dedup` | Deduplication via sync.Map | Scrape same video ID twice | Second returns Err "duplicate" |
| `TestYouTubeScraper_SearchVideos_NoAPIKey` | Empty API key | `NewYouTubeScraper("", nil)` | Returns Err "API key required" |
| `TestYouTubeScraper_SearchVideos_QuotaExhausted` | 403 response | httptest.Server returns 403 | Returns ErrQuotaExhausted |
| `TestYouTubeScraper_SearchVideos_Success` | Parses search response | httptest.Server with valid JSON | Returns []VideoMeta with correct fields |
| `TestYouTubeScraper_Scrape_CancelledContext` | Respects context cancellation | Cancel context immediately | Channel closes quickly, no results |
| `TestYouTubeScraper_ScrapeVideoIDs` | Scrape specific IDs | httptest.Server for transcript | Returns results for each ID |

---

### 1.12 `cmd/scraper-reddit/reddit` — Reddit Scraper

**Already tested:**
- FetchAll (type serialization, NewScraper, cancelled context)

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestScraper_FetchSubreddit_Success` | Full fetch with httptest | httptest.Server returning listing + comments | Posts with comments populated |
| `TestScraper_FetchSubreddit_429` | Rate limit response | httptest.Server returns 429 | Retry logic kicks in, eventually returns error |
| `TestScraper_FetchSubreddit_500` | Server error | httptest.Server returns 500 | Retry logic, returns error |
| `TestScraper_FetchComments_NoCommentListing` | Comment response with <2 listings | Response with single listing | Returns nil comments |
| `TestScraper_FetchComments_FiltersNonT1` | Non-comment children filtered | Mix of t1 and t3 kinds | Only t1 returned as comments |
| `TestScraper_HttpGet_UserAgent` | User-Agent header sent | httptest.Server checks header | Contains "wessley-scraper" |
| `TestScraper_HttpGet_NonOKStatus` | Unexpected status codes | httptest.Server returns 403 | Returns error |
| `TestScraper_CancelledContext` | Context cancellation | Cancel between subreddits | Returns partial results |

> **Note:** The current test can't override `baseURL` (hardcoded const). Refactoring to accept base URL in Config would make these tests possible. Document this as a prerequisite.

---

### 1.13 `cmd/scraper-sources/forums` — Forum Scraper

**Already tested:**
- parseSearchResults (dedup, source, title), DefaultForums, NewScraper

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestParseSearchResults_NoThreadLinks` | HTML without thread/topic links | Generic HTML | Returns empty slice |
| `TestParseSearchResults_AbsoluteURLs` | Links already absolute | `<a href="https://...">` | URL not prefixed with BaseURL |
| `TestParseSearchResults_EmptyTitle` | Link with empty text | `<a href="/threads/x">  </a>` | Skipped (empty title) |
| `TestParseSearchResults_MetadataKeywords` | Keywords in metadata | Query="brakes" | Metadata.Keywords contains forum name, "forum", query |
| `TestScraper_FetchForum_MaxPerForum` | Limits results | MaxPerForum=1, HTML has 5 links | Returns 1 post |
| `TestScraper_FetchAll_ContextCancelled` | Cancellation between forums | Cancel after first forum | Returns partial results |

---

### 1.14 `cmd/scraper-sources/ifixit` — iFixit Scraper

**Already tested:**
- buildGuideContent, extractFixes, NewScraper

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestBuildGuideContent_EmptySummary` | No summary | Guide with Summary="" | Content starts with steps |
| `TestBuildGuideContent_NoSteps` | Guide with empty steps | Guide with Steps=nil | Just summary |
| `TestBuildGuideContent_StepWithoutTitle` | Step with empty title | Step{Title: ""} | Shows "Step N:" without title |
| `TestExtractFixes_NoMatches` | Content with no fix keywords | "The weather is nice" | Returns nil |
| `TestExtractFixes_CaseInsensitive` | Uppercase fix words | "REPLACE and INSTALL" | Finds both |
| `TestScraper_FetchCategory_Success` | Full fetch with httptest | httptest.Server returning guide JSON | ScrapedPosts with correct fields |
| `TestScraper_FetchCategory_EmptyResponse` | Empty guide list | httptest.Server returns [] | Returns empty |

---

### 1.15 `cmd/scraper-sources/nhtsa` — NHTSA Scraper

**Already tested:**
- parseNHTSADate, extractSymptoms, NewScraper, FetchAll (type-level)

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestParseNHTSADate_AllFormats` | Each supported date format | "01/15/2024", "2024-01-15", RFC3339 | All parse to same date |
| `TestParseNHTSADate_Invalid` | Unparseable date | "not-a-date" | Returns zero time |
| `TestExtractSymptoms_NoMatch` | No symptom keywords | "Everything is fine" | Returns nil |
| `TestExtractSymptoms_AllSymptoms` | All 14 known symptoms | Text with all symptoms | All 14 found |
| `TestScraper_FetchMake_MaxPerMake` | Result limiting | MaxPerMake=1, 5 complaints | Returns 1 |
| `TestScraper_FetchMake_VehicleFormat` | Vehicle string formatting | year=2020, make="TOYOTA", model="CAMRY" | "2020 TOYOTA CAMRY" |
| `TestScraper_FetchMake_Success` | Full fetch with httptest | httptest.Server returning complaint JSON | ScrapedPosts with correct fields |

---

### 1.16 `cmd/api` — HTTP API Server

**Already tested:**
- HealthEndpoint, ChatEndpoint_EmptyQuestion, ChatEndpoint_InvalidJSON, LoadConfig_Defaults, EnvOr

**Missing tests:**

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `TestChatEndpoint_Success` | Full happy path | Mock RAG service returning Answer | 200, JSON with answer/sources/model/tokens |
| `TestChatEndpoint_RAGError` | RAG service failure | Mock RAG service returns error | 500, error JSON |
| `TestChatEndpoint_WithVehicle` | Vehicle passed through | Request with vehicle field | ragSvc.Query called with vehicle |
| `TestChatEndpoint_ContentType` | Response content type | Any valid request | Content-Type: application/json |
| `TestHealthEndpoint_ContentType` | Health response type | GET /api/health | Content-Type: application/json |
| `TestSemanticAdapter_Search` | Adapter delegates to VectorStore | Mock VectorStore | SearchFiltered called with correct args |
| `TestGraphAdapter_FindRelatedComponents` | Adapter combines FindByType + Neighbors | Mock GraphStore | Returns components and edges |
| `TestGraphAdapter_NoResults` | No components found | Mock returns empty | Returns empty slices |
| `TestLoadConfig_EnvOverrides` | Environment variables override defaults | Set PORT, ML_WORKER_URL etc. | Config reflects env values |
| `TestMiddlewareChain_Integration` | Full middleware chain | Request through Recover+Logger+CORS | All middleware applied, handler called |

> **Note:** To test `handleChat` with a real RAG service, extract `ragSvc` interface or make Service fields mockable. Currently `*rag.Service` with unexported embed/chat fields makes mocking difficult. The test should inject mocks at the `Service` struct level (embed/chat/search/graph fields).

---

### 1.17 `ml-worker` (Python) — ML gRPC Services

**Test framework:** `pytest` + `grpcio-testing` or `unittest.mock`

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `test_chat_servicer_builds_messages_system_prompt` | System prompt in messages | Request with system_prompt | First message is system role |
| `test_chat_servicer_builds_messages_context` | Context as user messages | Request with 2 context strings | 2 user messages before the main message |
| `test_chat_servicer_builds_messages_no_system` | No system prompt | Request without system_prompt | No system message |
| `test_chat_servicer_chat_success` | Unary chat | Mock httpx.Client → Ollama JSON | Returns reply, tokens_used, model |
| `test_chat_servicer_chat_default_model` | Default model used | Request with empty model | Uses CHAT_MODEL env or "mistral" |
| `test_chat_servicer_chat_custom_model` | Custom model | Request with model="llama3" | Ollama called with model="llama3" |
| `test_chat_servicer_chat_options` | Temperature and max_tokens | Set both | Ollama options include both |
| `test_chat_servicer_chat_ollama_error` | Ollama HTTP error | Mock httpx raises | context.abort with INTERNAL |
| `test_chat_servicer_stream_success` | Streaming chat | Mock httpx stream → JSON lines | Yields ChatChunks, last has done=True |
| `test_chat_servicer_stream_error` | Ollama stream error | Mock httpx raises | context.abort with INTERNAL |
| `test_embed_servicer_embed_single` | Single text embedding | Real or mocked SentenceTransformer | Returns values and dimensions |
| `test_embed_servicer_embed_batch` | Batch embedding | 3 texts | Returns 3 EmbedResponses |
| `test_embed_servicer_embed_batch_empty` | Empty batch | texts=[] | Returns empty embeddings list |
| `test_embed_servicer_dimensions` | Dimension correctness | Embed "test" | dimensions == len(values) |
| `test_health_check` | Health service responds | Health stub check "" | SERVING |
| `test_health_check_services` | Named service health | Check "wessley.ml.v1.ChatService" | SERVING |
| `test_serve_signal_handling` | Graceful shutdown | Send SIGTERM | Server stops without error |

> **Mocking strategy:** Mock `httpx.Client` for Ollama calls. For SentenceTransformer, either use the real model (small, `all-MiniLM-L6-v2` is ~80MB) or mock `self._model.encode()`.

---

### 1.18 `web` (Next.js) — Frontend

**Test framework:** `vitest` + `@testing-library/react`

| Test Name | Validates | Setup/Mocks | Expected Behavior |
|---|---|---|---|
| `test_api_chat_success` | `chat()` function | Mock fetch → 200 JSON | Returns ChatResponse |
| `test_api_chat_error` | `chat()` error handling | Mock fetch → 500 | Throws "API error: 500" |
| `test_api_chatStream_yields_chunks` | `chatStream()` generator | Mock fetch → ReadableStream | Yields decoded chunks |
| `test_api_chatStream_error` | Stream error | Mock fetch → 500 | Throws error |
| `test_api_chatStream_no_body` | Missing response body | Mock fetch → no body | Throws "No response body" |
| `test_api_default_url` | Default API URL | No env var | Uses "http://localhost:8080" |
| `test_ChatInput_renders` | ChatInput component renders | Render with props | Input field and submit button visible |
| `test_ChatInput_submits` | Form submission | Type text, click send | onSubmit called with text |
| `test_ChatInput_empty_submit` | Empty submission prevented | Click send without text | onSubmit not called (or handles gracefully) |
| `test_ChatMessages_renders_messages` | Message list | Render with 3 messages | All 3 visible |
| `test_ChatMessages_empty` | No messages | Render with [] | Empty state or placeholder |
| `test_Message_user` | User message styling | Message with role="user" | User-styled bubble |
| `test_Message_assistant` | Assistant message styling | Message with role="assistant" | Assistant-styled bubble |
| `test_Sources_renders` | Sources component | 2 sources | Both source titles visible |
| `test_Sources_empty` | No sources | sources=[] | Nothing rendered or "No sources" |
| `test_Header_renders` | Header component | Render | Logo/title visible |
| `test_chat_page_integration` | Chat page loads | Render page | Input and message area present |

---

## 2. Integration Tests

### 2.1 API → RAG → ML Worker (End-to-End)

| Test Name | Validates | Dependencies | Expected Behavior |
|---|---|---|---|
| `TestE2E_ChatFlow` | Full request flow | Docker Compose (all services) | POST /api/chat → 200 with answer, sources, model |
| `TestE2E_ChatFlow_EmptyQuestion` | Input validation | API running | POST /api/chat with empty question → 400 |
| `TestE2E_HealthCheck` | All services healthy | Docker Compose | GET /api/health → 200 {"status": "ok"} |
| `TestE2E_ChatWithVehicle` | Vehicle-filtered query | All services + seeded data | Response includes vehicle-relevant sources |

**Setup:** Docker Compose up, seed Qdrant with test vectors, seed Neo4j with test components.

### 2.2 Scraper → Ingest → Qdrant + Neo4j

| Test Name | Validates | Dependencies | Expected Behavior |
|---|---|---|---|
| `TestIngest_FullPipeline` | End-to-end ingestion | Qdrant, Neo4j, ML Worker | ScrapedPost → stored in Qdrant (search returns it) and Neo4j (component exists) |
| `TestIngest_Dedup` | Deduplication | Same as above + Redis or in-memory dedup | Second ingest of same post is skipped |
| `TestIngest_DLQ` | Dead letter queue | NATS, intentionally fail embed | After 3 retries, message appears on DLQ subject |
| `TestIngest_NATS_Consumer` | NATS message consumption | NATS + all pipeline deps | Publish to engine.ingest → data appears in Qdrant |

### 2.3 Docker Compose Smoke Test

| Test Name | Validates | Dependencies | Expected Behavior |
|---|---|---|---|
| `TestSmoke_AllServicesStart` | All containers start | Docker Compose | All 8 services running (wessley, ml-worker, web, neo4j, qdrant, redis, nats, ollama) |
| `TestSmoke_HealthChecks` | Infrastructure health | Docker Compose | Neo4j, Qdrant health checks pass |
| `TestSmoke_ServiceConnectivity` | Services can reach each other | Docker Compose | API can reach ML Worker, Neo4j, Qdrant |
| `TestSmoke_WebReachable` | Frontend serves | Docker Compose | GET http://localhost:3000 → 200 |

### 2.4 gRPC Client ↔ ML Worker

| Test Name | Validates | Dependencies | Expected Behavior |
|---|---|---|---|
| `TestGRPC_Embed_SingleText` | Embed unary RPC | ML Worker running | Returns embedding vector with correct dimensions |
| `TestGRPC_EmbedBatch` | EmbedBatch RPC | ML Worker running | Returns N embeddings for N texts |
| `TestGRPC_Chat_Unary` | Chat unary RPC | ML Worker + Ollama | Returns reply with tokens_used |
| `TestGRPC_ChatStream` | Chat streaming RPC | ML Worker + Ollama | Receives chunks, last chunk has done=True |
| `TestGRPC_Health` | Health check RPC | ML Worker running | Returns SERVING |
| `TestGRPC_Reflection` | gRPC reflection | ML Worker running | Can list services via reflection |

### 2.5 Semantic Store Integration (Qdrant)

Already covered in `engine/semantic/store_test.go` (gated on `QDRANT_ADDR`). Extend with:

| Test Name | Validates | Dependencies | Expected Behavior |
|---|---|---|---|
| `TestQdrant_HighDimensional` | Real embedding dimensions | Qdrant | Upsert/search with 384-dim vectors (MiniLM) works |
| `TestQdrant_LargePayload` | Large content in payload | Qdrant | 10KB content string stored and retrieved |
| `TestQdrant_MultiFilterSearch` | Combined filters | Qdrant | Filter by source + vehicle returns correct subset |

---

## 3. Test Infrastructure

### 3.1 Makefile Targets

```makefile
# Existing
test:                    ## Run all unit tests
	go test ./...

# Proposed additions
test-unit:               ## Run unit tests only (no external deps)
	go test -short ./...

test-race:               ## Run tests with race detector
	go test -race ./...

test-cover:              ## Run with coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-integration:        ## Run integration tests (requires Docker deps)
	QDRANT_ADDR=localhost:6334 go test -run Integration ./...

test-e2e:                ## Run E2E tests (requires full Docker Compose)
	docker compose -f deploy/docker-compose.yml up -d
	sleep 30  # wait for services
	go test -run E2E ./test/e2e/...
	docker compose -f deploy/docker-compose.yml down

test-python:             ## Run ML worker tests
	cd ml-worker && pip install -r requirements-test.txt && pytest -v

test-web:                ## Run frontend tests
	cd web && npm test

test-all: test-unit test-python test-web  ## Run all test suites

lint:                    ## Run linters
	golangci-lint run ./...
	cd ml-worker && ruff check .
	cd web && npx eslint .
```

### 3.2 Docker Test Dependencies

For integration tests, use a `deploy/docker-compose.test.yml`:

```yaml
services:
  neo4j-test:
    image: neo4j:5-community
    ports: ["17687:7687"]
    environment:
      NEO4J_AUTH: neo4j/testpassword
    tmpfs: /data  # ephemeral

  qdrant-test:
    image: qdrant/qdrant:latest
    ports: ["16334:6334"]
    tmpfs: /qdrant/storage

  nats-test:
    image: nats:2-alpine
    ports: ["14222:4222"]
    command: ["--jetstream"]
    tmpfs: /data

  redis-test:
    image: redis:7-alpine
    ports: ["16379:6379"]
```

Run: `docker compose -f deploy/docker-compose.test.yml up -d`

### 3.3 CI Pipeline Recommendations

```yaml
# GitHub Actions
name: CI
on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - run: go test -race -coverprofile=coverage.out ./...
      - uses: codecov/codecov-action@v4

  python-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with: { python-version: "3.12" }
      - run: cd ml-worker && pip install -r requirements.txt -r requirements-test.txt
      - run: cd ml-worker && pytest -v --cov

  web-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: "20" }
      - run: cd web && npm ci && npm test

  integration-tests:
    runs-on: ubuntu-latest
    services:
      qdrant: { image: "qdrant/qdrant:latest", ports: ["6334:6334"] }
      neo4j:
        image: neo4j:5-community
        ports: ["7687:7687"]
        env: { NEO4J_AUTH: "neo4j/testpassword" }
      nats: { image: "nats:2-alpine", ports: ["4222:4222"] }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - run: QDRANT_ADDR=localhost:6334 go test -run Integration ./...
```

### 3.4 Coverage Targets

| Package | Current Coverage (est.) | Target |
|---|---|---|
| `pkg/fn` | ~90% | 95% |
| `pkg/mid` | ~70% | 90% |
| `pkg/natsutil` | ~40% (no real conn) | 60% unit, 90% with integration |
| `pkg/repo` | ~20% (construction only) | 80% with mocks |
| `pkg/resilience` | ~85% | 95% |
| `engine/domain` | ~80% | 95% |
| `engine/graph` | ~30% (helpers only) | 80% with mocks |
| `engine/semantic` | ~60% (integration) | 85% with mocks |
| `engine/ingest` | ~50% | 85% |
| `engine/rag` | ~75% | 90% |
| `engine/scraper` | ~30% | 75% |
| `cmd/api` | ~40% | 80% |
| `cmd/scraper-*` | ~25% | 60% |
| `ml-worker` | 0% | 80% |
| `web` | 0% | 70% |

**Overall Go target:** 80% line coverage

---

## 4. Priority Matrix

### P0 — Must Have for MVP

These tests protect core functionality and prevent shipping broken code.

| Package | Tests | Rationale |
|---|---|---|
| `engine/domain` | All validation tests (§1.6) | Input gate — blocks bad data |
| `engine/rag` | Query success/error paths (§1.10) | Core user-facing feature |
| `engine/ingest` | Pipeline stages, chunking (§1.9) | Data quality depends on this |
| `cmd/api` | Handler tests (§1.16) | API contract |
| `pkg/resilience` | Circuit breaker lifecycle, rate limiter (§1.5) | Production stability |
| `pkg/fn` | Result/Collect/Pipeline (§1.1) | Foundation for all pipelines |
| `ml-worker` | Chat & Embed servicers (§1.17) | ML pipeline correctness |
| Integration | Docker Compose smoke test (§2.3) | Deployment confidence |
| Integration | gRPC client ↔ ML Worker (§2.4) | Cross-service contract |

### P1 — Important

| Package | Tests | Rationale |
|---|---|---|
| `engine/graph` | Mocked Neo4j operations (§1.7) | Knowledge graph correctness |
| `engine/semantic` | Mocked Qdrant operations (§1.8) | Vector search correctness |
| `engine/scraper` | Transcript + YouTube with httptest (§1.11) | Data ingestion quality |
| `pkg/mid` | All middleware tests (§1.2) | HTTP layer robustness |
| `pkg/repo` | Mocked CRUD operations (§1.4) | Generic repo correctness |
| `cmd/scraper-*` | HTTP-mocked scraper tests (§1.12-1.15) | Scraper reliability |
| `web` | API client + key components (§1.18) | Frontend reliability |
| Integration | Full ingest pipeline (§2.2) | End-to-end data flow |

### P2 — Nice to Have

| Package | Tests | Rationale |
|---|---|---|
| `pkg/fn` | Edge cases (empty, panic, single-item) (§1.1) | Robustness |
| `pkg/natsutil` | Additional carrier tests (§1.3) | Trace propagation |
| Integration | E2E chat flow (§2.1) | Full system validation |
| Integration | Extended Qdrant tests (§2.5) | Scale validation |
| `web` | Full page integration tests (§1.18) | UI regression |
| CI | Coverage enforcement in pipeline (§3.3) | Quality gate |

---

### Implementation Order (Recommended)

1. **Week 1:** P0 unit tests — domain validation, RAG, ingest stages, API handlers, resilience
2. **Week 2:** P0 ml-worker tests + P1 mocked store tests (graph, semantic, repo)
3. **Week 3:** P1 scraper tests + middleware + web tests
4. **Week 4:** Integration tests (Docker Compose smoke, gRPC, full pipeline)
5. **Ongoing:** P2 edge cases, coverage improvement, CI pipeline

---

*This plan covers ~200 test cases across Go, Python, and TypeScript. Each entry is implementation-ready.*
