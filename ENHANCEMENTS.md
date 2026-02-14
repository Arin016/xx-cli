# xx-cli — RAG & Systems-Level Enhancements

A prioritized list of staff/principal-engineer-level improvements to the RAG pipeline and overall system architecture. Each enhancement includes the what, why, how, and which files it touches.

---

## 1. ✅ Auto-Learning from Successful Commands (Online Learning)

**Priority:** P0 — highest impact, zero user effort — **DONE**

**What:** After a command executes successfully, automatically embed the prompt+command pair and append it to the vector store in the background. Before appending, check if a near-duplicate already exists (cosine similarity > 0.95) to prevent bloat.

**Implementation:**
- `spawnAutoLearn()` in `cmd/run.go` forks a detached `xx _learn <prompt> <command> <category>` subprocess via `os.Executable()` + `exec.Command` with nil stdin/stdout/stderr
- Hidden `xx _learn` subcommand in `cmd/autolearn.go` calls `rag.LearnFromSuccess()` which embeds, dedup-checks, and appends
- `rag.LearnFromSuccess()` in `internal/rag/rag.go` orchestrates: embed → load store → `HasNearDuplicate()` check → `Append()` if novel
- Wired into 3 places in `run.go`: after successful single command, after successful retry, after each successful workflow step
- Fire-and-forget design: `cmd.Start()` without `cmd.Wait()` — OS reaps the orphan. Zero latency impact on user

**Why:** The vector store gets smarter with every use without the user ever running `xx index` again. This is the difference between a static knowledge base and a self-improving system. In an interview, this demonstrates understanding of online learning, write-behind patterns, and deduplication strategies.

**How:**
1. After successful execution in `run.go`, spawn a goroutine that:
   - Embeds the text: `"'{prompt}' was successfully executed as: {command}"`
   - Loads the current store
   - Searches for existing docs with similarity > 0.95
   - If no near-duplicate → appends the new document and saves
   - If duplicate exists → skips silently
2. The goroutine runs async — user never waits for the embedding (~200ms)
3. Use a file lock (flock) to prevent concurrent writes if multiple xx instances run simultaneously

**Complexity:** Medium
**Files:** `cmd/run.go`, `internal/rag/store.go`, `internal/rag/rag.go`

**Interview talking points:**
- Write-behind pattern (async persistence after user-facing operation completes)
- Deduplication via semantic similarity (not exact string match)
- File-level concurrency control with flock
- Graceful degradation (if embedding fails, user experience is unaffected)

---

## 2. ✅ Incremental Append (O(1) Writes)

**Priority:** P0 — required for auto-learning to be efficient — **DONE**

**What:** Add an `Append()` method to the Store that writes a single document to the end of the binary file and updates the document count header, without rewriting the entire file.

**Implementation:**
- `Store.Append()` in `internal/rag/store.go` — opens file in read-write mode, seeks to end, writes doc in binary format, seeks back to byte 0, overwrites count with count+1
- `Store.HasNearDuplicate()` — loads store, computes cosine similarity against all docs, returns true if any > threshold
- `storePath` changed from function to `var storePath = func() string {...}` for test overridability
- Tested: doc count increments on novel commands, stays stable on repeats (dedup works)

**Why:** The current `Save()` rewrites all documents every time — O(n). For online learning where we add one doc per successful command, we need O(1) appends. At 1000+ docs, rewriting the entire file on every command would add noticeable latency.

**How:**
1. Open the file in read-write mode
2. Read the current document count from the first 4 bytes
3. Seek to end of file
4. Write the new document in the same binary format
5. Seek back to byte 0 and overwrite the count with count+1
6. Close the file

**Binary format reminder (from store.go):**
```
[4 bytes] document count (uint32)
[...documents...]
```

Each document:
```
[4 bytes] text length → [N bytes] text
[4 bytes] source length → [N bytes] source
[4 bytes] category length → [N bytes] category
[4 bytes] vector dimension → [dim*4 bytes] vector
```

**Complexity:** Low
**Files:** `internal/rag/store.go`

**Interview talking points:**
- Amortized O(1) append vs O(n) full rewrite
- Header-update pattern (count at byte 0)
- Why this is safe: single-writer with flock, append-only data section

---

## 3. ✅ Embedding Cache (LRU)

**Priority:** P1 — saves ~200ms per repeated query — **DONE**

**What:** Cache recent query embeddings in an in-memory LRU cache. If the same query was embedded recently, reuse the cached vector instead of calling Ollama again.

**Implementation:**
- Added `cache map[string]cachedEmbedding` and `cacheOrder []string` to `EmbedClient`
- `Embed()` checks cache before calling Ollama API — cache hit returns in 0ms vs ~200ms
- LRU eviction: oldest entry dropped when cache exceeds 100 entries
- `touchLRU()` moves accessed keys to back of order slice
- Memory budget: 100 entries × 768 floats × 4 bytes = ~300KB — negligible

**Why:** Users often repeat queries ("is chrome running", "show disk usage"). Each embedding call takes ~200ms. An LRU cache with 100 entries eliminates redundant API calls. This is the same pattern used by DNS resolvers and CDN edge caches.

**How:**
1. Add a `cache map[string][]float32` to `EmbedClient` with a max size of 100
2. Before calling Ollama, check if the exact query string is in the cache
3. On cache hit → return the cached vector (0ms vs 200ms)
4. On cache miss → call Ollama, store result in cache, evict oldest if full
5. Cache is in-memory only — no persistence needed (it's per-process)

**Advanced variant:** Use a similarity-based cache where we check if any cached query has cosine similarity > 0.99 with the new query. This catches near-duplicates like "is chrome running" vs "is chrome running?" — but adds O(cache_size) comparison cost per lookup.

**Complexity:** Low
**Files:** `internal/rag/embeddings.go`

**Interview talking points:**
- LRU eviction policy and why it's appropriate here (temporal locality)
- Trade-off: exact-match cache (O(1) lookup) vs similarity-based cache (O(n) but catches near-dupes)
- Memory budget: 100 entries × 768 floats × 4 bytes = ~300KB — negligible

---

## 4. ✅ Adaptive Relevance Scoring (Reinforcement Signal)

**Priority:** P1 — makes retrieval quality improve over time — **DONE**

**What:** Track a success/failure count per document. When a retrieved document leads to a successful command, increment its score. When it leads to a failure, decrement. Use this score as a multiplier on cosine similarity during search.

**Implementation:**
- Added `SuccessCount int32` and `FailureCount int32` to `Document` struct
- Binary format upgraded to v2 with version header — backward compatible with v1 (scoring fields default to 0)
- `adaptiveScore()` formula: `cosine * (1 + ln(1+successes) - 0.5*ln(1+failures))` with 0.01 floor
- `UpdateScore()` finds the most similar doc (above 0.5 threshold) and increments success/failure count
- `RecordFeedback()` in `rag.go` orchestrates: embed query → load store → update score → save
- `spawnFeedback()` in `run.go` forks detached `xx _feedback <prompt> <success|failure>` subprocess
- Hidden `_feedback` subcommand in `autolearn.go` calls `RecordFeedback()`
- Fires after every command execution (both success and failure) — zero latency impact

**Why:** Not all documents are equally reliable. A command that worked 10 times is more trustworthy than one that worked once. This is a lightweight form of bandit-style exploration/exploitation — the system learns which commands are reliable on this specific machine.

**How:**
1. Add `SuccessCount int32` and `FailureCount int32` fields to the `Document` struct
2. Update the binary format to include these two int32s per document
3. After command execution, find the document that was most relevant to this query and update its counts
4. In `Search()`, compute final score as: `cosineSimilarity * (1 + log(1 + successCount) - 0.5 * log(1 + failureCount))`
5. The log dampening prevents a single document from dominating forever

**Score formula explained:**
```
finalScore = cosine * (1 + ln(1 + successes) - 0.5 * ln(1 + failures))
```
- A doc with 10 successes and 0 failures: cosine * 3.4
- A doc with 10 successes and 5 failures: cosine * 2.5
- A doc with 0 successes and 5 failures: cosine * 0.1
- A brand new doc: cosine * 1.0 (neutral)

**Complexity:** Medium
**Files:** `internal/rag/store.go`, `internal/rag/rag.go`, `cmd/run.go`

**Interview talking points:**
- Multi-armed bandit analogy (explore new commands vs exploit known-good ones)
- Log dampening to prevent runaway scores
- Cold-start problem: new documents start at neutral (1.0 multiplier)
- This is the same principle behind Reddit's ranking algorithm and Hacker News's story scoring

---

## 5. TTL / Staleness Eviction

**Priority:** P2 — prevents unbounded growth

**What:** Add a `CreatedAt` timestamp to each document. During `xx index` rebuilds, evict history-sourced documents older than 90 days. Builtin and learned documents are never evicted.

**Why:** Command patterns change over time. If you switched from `npm` to `pnpm` 3 months ago, old `npm` history entries are noise that dilutes retrieval quality. TTL eviction keeps the store lean and relevant.

**How:**
1. Add `CreatedAt int64` (Unix timestamp) to the `Document` struct
2. Update binary format to include the timestamp
3. In `IndexAll()`, skip history entries older than 90 days
4. In `Append()` (for online learning), always set `CreatedAt` to `time.Now().Unix()`
5. Builtin docs get `CreatedAt = 0` (sentinel value meaning "never expires")

**Complexity:** Low
**Files:** `internal/rag/store.go`, `internal/rag/indexer.go`

**Interview talking points:**
- TTL eviction is the same pattern used by Redis, DNS caches, and CDN edge nodes
- Sentinel value (0) for immortal entries — avoids a separate "type" field
- Trade-off: 90 days is a heuristic. Could make it configurable via `config.json`

---

## 6. Contextual Re-ranking (Two-Stage Retrieval)

**Priority:** P2 — production-grade retrieval quality

**What:** After the initial cosine similarity search returns top-K candidates, apply a second-pass re-ranker that considers recency, frequency, and context match (project type, directory).

**Why:** Cosine similarity alone doesn't capture everything. A command that worked yesterday in this exact project is more relevant than one that worked 2 months ago in a different directory. Two-stage retrieval is the standard pattern in production search systems (Google, Elasticsearch, Pinecone all do this).

**How:**
1. Stage 1: Cosine similarity search returns top-10 candidates (cheap, O(n))
2. Stage 2: Re-rank the 10 candidates using a weighted score:
   ```
   finalScore = 0.6 * cosineSimilarity
              + 0.2 * recencyScore      // exponential decay: e^(-days_old / 30)
              + 0.1 * frequencyScore    // log(1 + use_count) / log(1 + max_use_count)
              + 0.1 * contextScore      // 1.0 if same project type, 0.5 if same category, 0.0 otherwise
   ```
3. Return top-5 after re-ranking

**Complexity:** Medium
**Files:** `internal/rag/rag.go`, `internal/rag/store.go`

**Interview talking points:**
- Two-stage retrieval: cheap recall (vector search) → expensive precision (re-ranking)
- Feature engineering for the re-ranker (recency, frequency, context)
- Why not use a learned re-ranker? At our scale, hand-tuned weights are simpler and more interpretable
- Same architecture as Google's search: inverted index → BERT re-ranker

---

## 7. Approximate Nearest Neighbor (LSH Index)

**Priority:** P3 — only needed at 10K+ documents

**What:** Build a Locality-Sensitive Hashing (LSH) index over the vectors to reduce search from O(n) to O(n/k). Use random hyperplane hashing to bucket vectors, then only search within the matching bucket.

**Why:** Brute-force cosine similarity is fine for ~100 docs. At 10K+ docs (heavy users over months), search latency becomes noticeable. LSH gives us sub-linear search time with minimal accuracy loss.

**How:**
1. Generate `h` random hyperplanes (768-dim vectors with random ±1 components)
2. For each document vector, compute its hash: for each hyperplane, the bit is 1 if dot(vector, hyperplane) > 0, else 0
3. Store documents in buckets keyed by their hash
4. At query time, hash the query vector and only search documents in the same bucket (and adjacent buckets for recall)
5. Typical config: h=8 hyperplanes → 256 buckets → ~40 docs per bucket at 10K total

**Complexity:** High
**Files:** `internal/rag/store.go` (new LSH index), `internal/rag/rag.go`

**Interview talking points:**
- LSH theory: random hyperplanes preserve cosine similarity with high probability (Johnson-Lindenstrauss lemma)
- Trade-off: more hyperplanes = more buckets = faster search but lower recall
- Multi-probe LSH: check adjacent buckets (Hamming distance 1) to recover recall
- Why not HNSW or IVF? LSH is simpler to implement from scratch and sufficient for our scale
- This is the same algorithm used by Spotify for music recommendation and Shazam for audio fingerprinting

---

## 8. Vector Store Compaction

**Priority:** P3 — maintenance optimization

**What:** Periodic compaction that merges near-duplicate documents (similarity > 0.98), keeping the one with the highest success count. Run automatically during `xx index` or as a separate `xx compact` command.

**Why:** Over time, online learning accumulates slight variations of the same knowledge ("how much RAM" vs "how much memory" vs "total RAM on this mac"). Compaction merges these into a single high-quality entry, keeping the store lean.

**How:**
1. Load all documents into memory
2. For each pair of documents with similarity > 0.98:
   - Keep the one with higher success count
   - Merge their metadata (sum success counts, keep the more recent timestamp)
   - Delete the other
3. Rewrite the store with the compacted set
4. Report: "Compacted 150 → 120 documents (30 duplicates merged)"

**Complexity:** Medium (O(n²) pairwise comparison, but n is small)
**Files:** `internal/rag/store.go`, `cmd/index.go` (or new `cmd/compact.go`)

**Interview talking points:**
- Same concept as LSM-tree compaction in LevelDB/RocksDB
- O(n²) is fine for n < 10K; for larger stores, use LSH to find near-duplicate candidates first
- Idempotent operation — running it twice produces the same result

---

## Implementation Order

| Phase | Enhancement | Effort | Impact | Status |
|-------|------------|--------|--------|--------|
| 1 | Incremental Append (O(1) writes) | 1 hour | Enables everything else | ✅ Done |
| 1 | Auto-Learning from Successful Commands | 2 hours | Highest user-facing impact | ✅ Done |
| 2 | Embedding Cache (LRU) | 1 hour | 200ms savings per repeated query | ✅ Done |
| 2 | TTL / Staleness Eviction | 1 hour | Prevents unbounded growth | |
| 3 | Adaptive Relevance Scoring | 2 hours | Retrieval quality improves over time | ✅ Done |
| 3 | Contextual Re-ranking | 2 hours | Production-grade retrieval | |
| 4 | LSH Index | 4 hours | Sub-linear search (only needed at scale) | |
| 4 | Vector Store Compaction | 2 hours | Maintenance optimization | |

**Total estimated effort:** ~15 hours across 4 phases.

Phase 1 is the foundation — everything else builds on incremental append and auto-learning. Phase 2 is quick wins. Phase 3 is where the system starts to feel genuinely intelligent. Phase 4 is future-proofing for scale.
