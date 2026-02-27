# Wessley AI — Intelligent Manual Ingestion Pipeline

## Current (Dumb) Pipeline
```
PDF → raw text → chunk by sentences → embed with Ollama → store in Qdrant
```
**Problems:**
- No understanding of document structure
- No entity extraction (components, specs, part numbers)
- No relationship extraction (what connects to what)
- Chunks are arbitrary, not semantically meaningful
- Graph gets flat Component nodes with no edges
- Can't answer "what parts does the 2019 Camry brake system use?"

## New (Intelligent) Pipeline

```
PDF
 │
 ▼
┌─────────────────────────────────────┐
│ STAGE 1: Document Intelligence      │
│ (Local Transformers — FREE)         │
│                                     │
│ • PDF → structured text extraction  │
│ • Page layout analysis              │
│ • Table extraction                  │
│ • Image/diagram detection           │
│ • Section hierarchy detection       │
│   (Chapter → Section → Subsection)  │
│                                     │
│ Models:                             │
│ • layoutlm-base (document layout)   │
│ • table-transformer (table extract) │
│ Tools: pdfplumber / PyMuPDF         │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ STAGE 2: NER + Entity Extraction    │
│ (Local Transformers — FREE)         │
│                                     │
│ Extract from each section:          │
│ • Components (ECU, relay, sensor)   │
│ • Part numbers (89661-06E71)        │
│ • Specifications (12V, 15A, 3.2Ω)  │
│ • Connector IDs (C101, E35)        │
│ • Wire colors (BLK/WHT, RED/GRN)   │
│ • Torque specs (25 N·m)            │
│ • Fluid types (DOT 4, ATF WS)     │
│ • Warning/caution flags             │
│                                     │
│ Models:                             │
│ • Custom NER fine-tuned on auto     │
│   (or dslim/bert-base-NER + rules) │
│ • Regex patterns for part numbers,  │
│   specs, connector IDs              │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ STAGE 3: Claude Intelligence        │
│ (Claude Agents SDK — SMART)         │
│                                     │
│ For each section, Claude:           │
│                                     │
│ A) CLASSIFY                         │
│    → System (Engine, Brakes, HVAC)  │
│    → Subsystem (Fuel Injection,     │
│      ABS, Compressor)               │
│    → Document type (Procedure,      │
│      Wiring Diagram, Specs Table,   │
│      Troubleshooting, Maintenance)  │
│                                     │
│ B) EXTRACT RELATIONSHIPS            │
│    → Component A --connects_to→ B   │
│    → Component A --powers→ B        │
│    → Component A --part_of→ System  │
│    → Procedure --requires→ Tool     │
│    → Symptom --caused_by→ Component │
│    → Component --has_spec→ Value    │
│                                     │
│ C) GENERATE STRUCTURED KNOWLEDGE    │
│    → Repair procedures as steps     │
│    → Diagnostic flowcharts          │
│    → Wiring paths (pin-to-pin)      │
│    → Maintenance intervals          │
│                                     │
│ D) CROSS-REFERENCE                  │
│    → Link to TSBs and recalls       │
│    → Link to known failure modes    │
│    → Link to related systems        │
│                                     │
│ Prompt structure:                   │
│ "You are an automotive electrical   │
│  engineer. Given this manual        │
│  section for a {year} {make}        │
│  {model}, extract all components,   │
│  their relationships, specs, and    │
│  connections. Output as structured  │
│  JSON."                             │
│                                     │
│ SDK: Claude Agents with tools:      │
│ • neo4j_query — check existing graph│
│ • add_relationship — create edges   │
│ • search_knowledge — RAG lookup     │
│ • flag_anomaly — mark conflicts     │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ STAGE 4: Graph Construction         │
│ (Neo4j — STRUCTURED)                │
│                                     │
│ Create/update nodes:                │
│ • Make → Model → Generation → Trim  │
│   → ModelYear                       │
│ • System → Subsystem → Component    │
│ • WiringPath (pin-to-pin traces)    │
│ • Procedure (repair steps)          │
│ • Specification (values + units)    │
│ • Tool (required tools)             │
│ • Fluid (required fluids)           │
│                                     │
│ Create relationships:               │
│ • HAS_SYSTEM, HAS_SUBSYSTEM         │
│ • CONTAINS_COMPONENT                │
│ • CONNECTS_TO (with wire info)      │
│ • POWERS, GROUNDS, SIGNALS          │
│ • REQUIRES_TOOL, REQUIRES_FLUID     │
│ • DIAGNOSED_BY (symptom→procedure)  │
│ • SUPERSEDED_BY (part replacement)  │
│ • SHARED_WITH (cross-vehicle parts) │
│                                     │
│ Properties on relationships:        │
│ • wire_color, pin_number            │
│ • torque_spec, voltage              │
│ • page_reference, manual_section    │
│ • confidence_score (from Claude)    │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ STAGE 5: Semantic Embedding         │
│ (Multi-Model — RICH)                │
│                                     │
│ Embed at multiple granularities:    │
│                                     │
│ A) Section-level (for RAG search)   │
│    → "How to replace brake pads on  │
│      2019 Camry" → finds the exact  │
│      procedure section              │
│    → Model: nomic-embed-text (768d) │
│                                     │
│ B) Component-level (for similarity) │
│    → Find similar components across │
│      vehicles ("what else uses the  │
│      same ECU?")                    │
│    → Model: nomic-embed-text        │
│                                     │
│ C) Procedure-level (for repair)     │
│    → Step-by-step procedures        │
│    → Troubleshooting flowcharts     │
│    → Model: nomic-embed-text        │
│                                     │
│ Qdrant collections:                 │
│ • wessley-sections (manual sections)│
│ • wessley-components (components)   │
│ • wessley-procedures (procedures)   │
│                                     │
│ Metadata filters:                   │
│ • make, model, year, system         │
│ • doc_type, confidence              │
└─────────────────────────────────────┘
```

## Technology Stack

### Local Transformers (Python, FREE, runs on CPU)
```python
# Document layout + table extraction
pip install pdfplumber pymupdf transformers torch

# NER + entity extraction  
# Option A: Fine-tuned automotive NER
# Option B: dslim/bert-base-NER + regex rules for part numbers/specs
from transformers import pipeline
ner = pipeline("ner", model="dslim/bert-base-NER", device=-1)  # CPU

# Sentence transformers for local embedding
# (backup to Ollama, faster for batch)
from sentence_transformers import SentenceTransformer
model = SentenceTransformer('nomic-ai/nomic-embed-text-v1.5')
```

### Claude Agents SDK (Intelligence Layer)
```python
import anthropic

# Agent with tools for graph interaction
agent = anthropic.Agent(
    model="claude-sonnet-4-20250514",
    tools=[
        neo4j_query_tool,      # Read existing graph
        add_node_tool,          # Create graph nodes
        add_relationship_tool,  # Create graph edges
        search_vectors_tool,    # RAG lookup
        flag_anomaly_tool,      # Mark data conflicts
    ],
    system="""You are an automotive electrical engineer and knowledge graph expert.
    Given a vehicle manual section, extract ALL:
    1. Components (name, type, part number, specs)
    2. Relationships (what connects to what, power flows, signal paths)
    3. Procedures (step-by-step repair/diagnostic instructions)
    4. Specifications (voltages, resistances, torques, fluid types)
    
    Use the tools to query the existing graph and add new knowledge.
    Flag any conflicts with existing data.
    Output confidence scores for each extraction."""
)
```

### Processing Budget
- **Transformers (local):** FREE — CPU inference, ~2 sec/page
- **Claude:** ~$0.003/page (sonnet, ~500 tokens in, ~1000 out per section)
- **For 1000 manuals × ~200 pages avg:** ~$600 total for Claude
- **Strategy:** Use transformers for 90% of extraction, Claude only for complex sections (wiring diagrams, troubleshooting trees, ambiguous classifications)

## Pipeline Runner Architecture

```
cmd/manual-ingester/main.go (Go orchestrator)
    │
    ├── Calls Python transformer worker (HTTP or subprocess)
    │   └── scripts/manual_worker.py
    │       ├── PDF extraction (pdfplumber)
    │       ├── Section detection (layout analysis)
    │       ├── NER (transformers)
    │       └── Returns structured JSON
    │
    ├── Calls Claude (via Anthropic SDK) for complex sections
    │   └── Only sections flagged as:
    │       - Wiring diagrams
    │       - Troubleshooting flowcharts
    │       - Ambiguous classification (confidence < 0.7)
    │       - Tables with specs
    │
    ├── Writes to Neo4j (graph construction)
    │   └── engine/graph/enricher.go (extended)
    │
    └── Writes to Qdrant (vector embedding)
        └── Multi-collection: sections, components, procedures
```

## Output Per Manual

For a single 200-page vehicle manual, the pipeline produces:

### Neo4j Nodes (~50-200 per manual)
- 1 ModelYear node (linked to Make→Model)
- 8-15 System nodes (Engine, Brakes, HVAC, Electrical, etc.)
- 20-40 Subsystem nodes
- 50-150 Component nodes (with specs as properties)
- 10-30 Procedure nodes
- 5-15 Specification nodes

### Neo4j Relationships (~200-800 per manual)
- HAS_SYSTEM, HAS_SUBSYSTEM, CONTAINS_COMPONENT
- CONNECTS_TO, POWERS, GROUNDS, SIGNALS
- DIAGNOSED_BY, REQUIRES_TOOL
- DOCUMENTED_IN (back-reference to manual page)

### Qdrant Vectors (~100-500 per manual)
- Section embeddings (for RAG: "how to replace X on Y")
- Component embeddings (for similarity search)
- Procedure embeddings (for repair lookup)

## Example: Processing a Brake System Section

**Input (from PDF):**
```
BRAKE SYSTEM - HYDRAULIC
The 2019 Camry uses a hydraulic brake system with ABS.

Master Cylinder: Part# 47201-06350
  - Bore: 23.81mm
  - Fluid: DOT 3 or DOT 4

Front Brake Caliper: Part# 47750-06310
  - Piston diameter: 57.15mm
  - Torque: Caliper bolt 107 N·m

ABS Actuator: Part# 44050-06520
  - Connected to: ECU via CAN bus (pins 1-4)
  - Fuse: EFI 30A (fuse box engine room)
```

**Stage 2 output (Transformers NER):**
```json
{
  "components": [
    {"name": "Master Cylinder", "part_number": "47201-06350", "type": "hydraulic"},
    {"name": "Front Brake Caliper", "part_number": "47750-06310", "type": "mechanical"},
    {"name": "ABS Actuator", "part_number": "44050-06520", "type": "electronic"}
  ],
  "specs": [
    {"component": "Master Cylinder", "property": "bore", "value": "23.81", "unit": "mm"},
    {"component": "Front Brake Caliper", "property": "torque", "value": "107", "unit": "N·m"},
    {"component": "ABS Actuator", "property": "fuse", "value": "30", "unit": "A"}
  ],
  "fluids": [{"name": "DOT 3", "system": "brake"}, {"name": "DOT 4", "system": "brake"}]
}
```

**Stage 3 output (Claude):**
```json
{
  "system": "Brakes",
  "subsystem": "Hydraulic Brake System",
  "relationships": [
    {"from": "Master Cylinder", "to": "Front Brake Caliper", "type": "HYDRAULIC_PRESSURE"},
    {"from": "ABS Actuator", "to": "ECU", "type": "SIGNALS", "protocol": "CAN", "pins": "1-4"},
    {"from": "ABS Actuator", "to": "EFI Fuse 30A", "type": "POWERED_BY"},
    {"from": "Master Cylinder", "to": "Hydraulic Brake System", "type": "PART_OF"},
    {"from": "Front Brake Caliper", "to": "Hydraulic Brake System", "type": "PART_OF"},
    {"from": "ABS Actuator", "to": "Hydraulic Brake System", "type": "PART_OF"}
  ],
  "confidence": 0.95
}
```

## Implementation Priority

### Phase 1: Python Worker (Week 1)
- PDF extraction with pdfplumber
- Section detection (heading patterns, page structure)
- Regex NER (part numbers, specs, connectors)
- Output structured JSON per section
- Basic sentence-transformer embedding

### Phase 2: Claude Intelligence (Week 2)
- Claude Agents SDK integration
- Relationship extraction for complex sections
- Wiring diagram interpretation
- Troubleshooting tree extraction
- Cross-reference with existing graph

### Phase 3: Graph Enrichment (Week 3)
- Extended Neo4j schema (Procedure, Specification, WiringPath nodes)
- Multi-collection Qdrant setup
- Backfill existing ManualEntry nodes
- Quality scoring and validation
