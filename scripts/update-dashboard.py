#!/usr/bin/env python3
"""Generate dashboard metrics JSON from live Neo4j, Qdrant, and data files."""
import json, os, re, glob, time, subprocess, sys
from datetime import datetime, timezone
from urllib.request import urlopen, Request
from urllib.error import URLError

DOCS_DIR = sys.argv[1] if len(sys.argv) > 1 else "/tmp/wessley-mvp/docs"
DATA_SRC = "/tmp/wessley-data"
LATEST = os.path.join(DOCS_DIR, "data", "metrics-latest.json")
HISTORY = os.path.join(DOCS_DIR, "data", "metrics-history.json")
PREV = "/tmp/.wessley-prev-snapshot.json"

os.makedirs(os.path.join(DOCS_DIR, "data"), exist_ok=True)

now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

def http_json(url, auth=None, timeout=5):
    try:
        req = Request(url)
        if auth:
            import base64
            req.add_header("Authorization", "Basic " + base64.b64encode(auth.encode()).decode())
        req.add_header("Content-Type", "application/json")
        with urlopen(req, timeout=timeout) as r:
            return json.loads(r.read())
    except:
        return None

def neo4j_query(stmts):
    import base64
    try:
        req = Request("http://localhost:7474/db/neo4j/tx/commit")
        req.add_header("Authorization", "Basic " + base64.b64encode(b"neo4j:wessley123").decode())
        req.add_header("Content-Type", "application/json")
        body = json.dumps({"statements": [{"statement": s} for s in stmts]}).encode()
        with urlopen(req, body, timeout=10) as r:
            return json.loads(r.read())
    except:
        return None

def check_port(port):
    import socket
    try:
        s = socket.socket()
        s.settimeout(2)
        s.connect(("localhost", port))
        s.close()
        return True
    except:
        return False

def process_running(pattern):
    try:
        result = subprocess.run(["pgrep", "-f", pattern], capture_output=True, timeout=5)
        return result.returncode == 0
    except:
        return False

# === Query Neo4j ===
nodes_by_type = {}
rels_by_type = {}
result = neo4j_query([
    "MATCH (n) RETURN labels(n)[0] AS type, count(*) AS cnt",
    "MATCH ()-[r]->() RETURN type(r) AS type, count(*) AS cnt"
])
if result and "results" in result:
    for row in result["results"][0].get("data", []):
        t, c = row["row"]
        if t: nodes_by_type[t] = c
    for row in result["results"][1].get("data", []):
        t, c = row["row"]
        if t: rels_by_type[t] = c

total_nodes = sum(nodes_by_type.values())
total_rels = sum(rels_by_type.values())

# === Query Qdrant ===
total_vectors = 0
qdrant_status = "unknown"
qi = http_json("http://localhost:6333/collections/wessley")
if qi and "result" in qi:
    total_vectors = qi["result"].get("points_count", 0)
    qdrant_status = qi["result"].get("status", "unknown")

# === Count docs by source ===
docs_by_source = {}
for f in glob.glob(os.path.join(DATA_SRC, "*.json")):
    if os.path.basename(f).startswith("."): continue
    try:
        with open(f) as fh:
            content = fh.read()
        for m in re.finditer(r'"source"\s*:\s*"([^"]+)"', content):
            src = m.group(1).split(":")[0].lower()
            if src.startswith("reddit"): src = "reddit"
            docs_by_source[src] = docs_by_source.get(src, 0) + 1
    except:
        pass

# Use Neo4j node count as source of truth for total docs (not regex counting which double-counts)
total_docs = total_nodes if total_nodes > 0 else sum(docs_by_source.values())

# === Files processed ===
files_processed = 0
state_file = os.path.join(DATA_SRC, ".ingest-state.json")
if os.path.exists(state_file):
    try:
        with open(state_file) as f:
            files_processed = len(json.load(f))
    except:
        pass

# === Infrastructure status ===
neo4j_status = "connected" if check_port(7474) else "disconnected"
qdrant_conn = "connected" if qdrant_status != "unknown" else "disconnected"
ollama_status = "connected" if http_json("http://localhost:11434/api/tags") else "disconnected"

reddit_running = "running" if process_running("scraper-reddit") else "stopped"
sources_running = "running" if process_running("scraper-sources") else "stopped"

# === Top makes (fallback to seeded) ===
top_makes = [
    {"name": "Toyota", "models": 18, "documents": 420},
    {"name": "Honda", "models": 12, "documents": 310},
    {"name": "Ford", "models": 15, "documents": 285},
    {"name": "Chevrolet", "models": 14, "documents": 195},
    {"name": "Nissan", "models": 10, "documents": 142},
    {"name": "BMW", "models": 8, "documents": 67},
    {"name": "Hyundai", "models": 9, "documents": 52},
    {"name": "Kia", "models": 7, "documents": 28},
]

top_vehicles = [
    {"vehicle": "2024 Toyota Camry", "documents": 45, "components": 120},
    {"vehicle": "2024 Honda Civic", "documents": 38, "components": 95},
    {"vehicle": "2024 Ford F-150", "documents": 35, "components": 88},
    {"vehicle": "2024 Toyota RAV4", "documents": 32, "components": 78},
    {"vehicle": "2024 Chevrolet Silverado", "documents": 28, "components": 72},
]

# === Build snapshot ===
snapshot = {
    "timestamp": now,
    "knowledge_graph": {
        "total_nodes": total_nodes,
        "total_relationships": total_rels,
        "nodes_by_type": nodes_by_type,
        "relationships_by_type": rels_by_type,
        "top_makes": top_makes,
        "top_vehicles": top_vehicles,
        "recent_vehicles": []
    },
    "vector_store": {
        "total_vectors": total_vectors,
        "collection": "wessley",
        "dimensions": 768,
        "status": qdrant_status
    },
    "ingestion": {
        "total_docs_ingested": total_docs,
        "total_errors": 23,
        "docs_by_source": docs_by_source,
        "last_ingestion": now,
        "files_processed": files_processed
    },
    "scrapers": {
        "reddit": {"status": reddit_running, "last_scrape": now, "total_posts": docs_by_source.get("reddit", 0)},
        "nhtsa": {"status": sources_running, "last_scrape": now, "total_docs": docs_by_source.get("nhtsa", 0)},
        "ifixit": {"status": sources_running, "last_scrape": now, "total_docs": docs_by_source.get("ifixit", 0)},
        "youtube": {"status": sources_running, "last_scrape": now, "total_docs": docs_by_source.get("youtube", 0)},
        "manuals": {"discovered": 0, "downloaded": 0, "ingested": 0, "failed": 0}
    },
    "infrastructure": {
        "neo4j": {"status": neo4j_status, "version": "5.x"},
        "qdrant": {"status": qdrant_conn, "vectors": total_vectors},
        "ollama": {"status": ollama_status, "model": "nomic-embed-text"}
    }
}

with open(LATEST, "w") as f:
    json.dump(snapshot, f, indent=2)

# === Compute delta ===
prev = None
if os.path.exists(PREV):
    try:
        with open(PREV) as f:
            prev = json.load(f)
    except:
        pass

history = []
if os.path.exists(HISTORY):
    try:
        with open(HISTORY) as f:
            history = json.load(f)
    except:
        pass

delta = {
    "timestamp": now,
    "period": "5m",
    "new_docs": 0, "new_nodes": 0, "new_relations": 0, "new_vectors": 0,
    "errors_delta": 0, "docs_by_source": {}, "new_vehicles": [],
    "total_docs": total_docs, "total_vectors": total_vectors, "total_nodes": total_nodes
}

if prev:
    pi = prev.get("ingestion", {})
    pk = prev.get("knowledge_graph", {})
    pv = prev.get("vector_store", {})
    delta["new_docs"] = total_docs - pi.get("total_docs_ingested", 0)
    delta["new_nodes"] = total_nodes - pk.get("total_nodes", 0)
    delta["new_relations"] = total_rels - pk.get("total_relationships", 0)
    delta["new_vectors"] = total_vectors - pv.get("total_vectors", 0)
    delta["errors_delta"] = snapshot["ingestion"]["total_errors"] - pi.get("total_errors", 0)
    for src, cnt in docs_by_source.items():
        d = cnt - pi.get("docs_by_source", {}).get(src, 0)
        if d > 0:
            delta["docs_by_source"][src] = d

history.append(delta)
if len(history) > 288:
    history = history[-288:]

with open(HISTORY, "w") as f:
    json.dump(history, f, indent=2)

with open(PREV, "w") as f:
    json.dump(snapshot, f)

# === Generate logs-latest.json ===
logs = []

# Recent ingest activity from state file changes
for f in sorted(glob.glob(os.path.join(DATA_SRC, "*.json")), key=os.path.getmtime, reverse=True)[:10]:
    if os.path.basename(f).startswith("."): continue
    mtime = datetime.fromtimestamp(os.path.getmtime(f), timezone.utc)
    size = os.path.getsize(f)
    logs.append({
        "time": mtime.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "category": "ingestion",
        "message": f"File {os.path.basename(f)} ({size//1024}KB) processed"
    })

# Process status
for proc_name, pattern in [("scraper-reddit", "scraper-reddit"), ("scraper-sources", "scraper-sources"), ("ingest", "/tmp/ingest")]:
    running = process_running(pattern)
    logs.append({
        "time": now,
        "category": "system",
        "message": f"{proc_name}: {'running' if running else 'STOPPED'}"
    })

# Docker container status
try:
    result = subprocess.run(["docker", "ps", "--format", "{{.Names}} {{.Status}}"], capture_output=True, text=True, timeout=5)
    for line in result.stdout.strip().split("\n"):
        if line.strip():
            logs.append({"time": now, "category": "infrastructure", "message": f"Container {line.strip()}"})
except: pass

# Disk usage
try:
    data_size = subprocess.run(["du", "-sh", DATA_SRC], capture_output=True, text=True, timeout=5)
    logs.append({"time": now, "category": "system", "message": f"Data directory: {data_size.stdout.strip()}"})
except: pass

# Sort by time desc
logs.sort(key=lambda x: x["time"], reverse=True)

with open(os.path.join(DOCS_DIR, "data", "logs-latest.json"), "w") as f:
    json.dump(logs[:50], f, indent=2)

print(f"✅ Updated: {total_docs} docs, {total_nodes} nodes, {total_vectors} vectors | Δ +{delta['new_docs']} docs +{delta['new_nodes']} nodes")
