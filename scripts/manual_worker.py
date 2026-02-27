#!/usr/bin/env python3
"""Intelligent manual ingestion — PDF → structured knowledge → JSON for Go ingest pipeline.

Usage: python3 manual_worker.py <pdf_path> <output_dir> [--make MAKE] [--model MODEL] [--year YEAR]
"""

import pdfplumber
import json
import os
import sys
import re
import hashlib
import logging
from pathlib import Path
from datetime import datetime, timezone
from typing import Optional

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
log = logging.getLogger("manual_worker")

# ── Regex NER patterns ──────────────────────────────────────────────────────

RE_PART_NUMBER = re.compile(r'\b[0-9]{4,5}[-–][0-9A-Z]{4,8}\b')
RE_VOLTAGE = re.compile(r'\b\d+(?:\.\d+)?\s*[Vv](?:olt(?:s)?)?\b')
RE_AMPERAGE = re.compile(r'\b\d+(?:\.\d+)?\s*[Aa](?:mp(?:s|ere(?:s)?)?)?\b')
RE_RESISTANCE = re.compile(r'\b\d+(?:\.\d+)?\s*(?:Ω|[Kk]Ω|ohm(?:s)?)\b')
RE_TORQUE = re.compile(r'\b\d+(?:\.\d+)?\s*(?:N[·.]m|ft[·.\-]?lb(?:f)?|kgf[·.]cm)\b')
RE_CONNECTOR = re.compile(r'\b[A-Z][0-9]{1,4}\b')
RE_WIRE_COLORS = re.compile(
    r'\b(?:BLK|WHT|RED|GRN|BLU|YEL|ORG|PNK|BRN|GRY|VIO|PUR|TAN|LT\s*GRN|LT\s*BLU|DK\s*GRN|DK\s*BLU)'
    r'(?:/(?:BLK|WHT|RED|GRN|BLU|YEL|ORG|PNK|BRN|GRY|VIO|PUR|TAN|LT\s*GRN|LT\s*BLU|DK\s*GRN|DK\s*BLU))?\b',
    re.IGNORECASE
)
RE_FLUID = re.compile(r'\b(?:DOT\s*[34]|ATF(?:\s*WS)?|CVT\s*fluid|[05]W[-–][23]0|75W[-–]90|GL[-–][45])\b', re.IGNORECASE)
RE_COMPONENT_TYPES = re.compile(
    r'\b(?:ECU|PCM|BCM|TCM|ABS|SRS|HVAC|sensor|relay|fuse|actuator|solenoid|motor|pump|valve|switch)\b',
    re.IGNORECASE
)

# Skip patterns for non-useful sections
SKIP_PATTERNS = re.compile(
    r'^\s*(?:table\s+of\s+contents|index|copyright|disclaimer|foreword|preface|how\s+to\s+use)',
    re.IGNORECASE
)

# Section heading detection
RE_ALL_CAPS_HEADING = re.compile(r'^[A-Z][A-Z\s\-/&]{4,}$', re.MULTILINE)
RE_NUMBERED_SECTION = re.compile(r'^\d+[\.\-]\d*\s+[A-Z]', re.MULTILINE)
RE_CHAPTER = re.compile(r'^(?:Chapter|Section|CHAPTER|SECTION)\s+\d+', re.MULTILINE)


def detect_vehicle_from_filename(pdf_path: str) -> dict:
    """Try to extract make/model/year from filename."""
    name = Path(pdf_path).stem.lower()
    # Common pattern: Make-Model-Year or Model-Year
    year_match = re.search(r'(19|20)\d{2}', name)
    year = int(year_match.group()) if year_match else 0

    known_makes = {
        'toyota': 'Toyota', 'honda': 'Honda', 'ford': 'Ford', 'chevrolet': 'Chevrolet',
        'nissan': 'Nissan', 'bmw': 'BMW', 'mercedes': 'Mercedes-Benz', 'hyundai': 'Hyundai',
        'kia': 'Kia', 'subaru': 'Subaru', 'mazda': 'Mazda', 'lexus': 'Lexus',
    }
    make = ""
    for key, val in known_makes.items():
        if key in name:
            make = val
            break

    # Model is whatever remains
    model = name
    for pat in [r'(19|20)\d{2}', r'repair[-_]?manual', r'service[-_]?manual', r'owner']:
        model = re.sub(pat, '', model, flags=re.IGNORECASE)
    for key in known_makes:
        model = model.replace(key, '')
    model = re.sub(r'[-_]+', ' ', model).strip()
    if not model:
        model = "Unknown"

    # For the test PDF: 3S-GTE-1991
    if '3s' in name and 'gte' in name:
        make = make or 'Toyota'
        model = '3S-GTE'

    return {"make": make or "Unknown", "model": model, "year": year}


def extract_sections(pdf_path: str, max_pages: int = 0) -> list:
    """Stage 1: Extract text and detect sections from PDF."""
    sections = []
    current_section = {"title": "Introduction", "content": "", "pages": [], "tables": []}

    log.info(f"Opening PDF: {pdf_path}")

    with pdfplumber.open(pdf_path) as pdf:
        total_pages = len(pdf.pages)
        log.info(f"Total pages: {total_pages}")

        page_limit = max_pages if max_pages > 0 else total_pages
        if page_limit < total_pages:
            log.info(f"Processing first {page_limit} of {total_pages} pages (--max-pages)")

        for i in range(min(page_limit, total_pages)):
            page = pdf.pages[i]
            page_num = i + 1
            if page_num % 25 == 0:
                log.info(f"  Processing page {page_num}/{total_pages}")

            text = page.extract_text() or ""

            # Extract tables
            tables = page.extract_tables() or []

            # Detect section breaks
            heading = None
            for line in text.split('\n')[:5]:  # Check first 5 lines
                line = line.strip()
                if not line:
                    continue
                if RE_ALL_CAPS_HEADING.match(line) and len(line) > 5:
                    heading = line.title()
                    break
                if RE_CHAPTER.match(line):
                    heading = line.strip()
                    break
                if RE_NUMBERED_SECTION.match(line):
                    heading = line.strip()
                    break

            if heading and heading != current_section["title"]:
                # Save current section if it has content
                if current_section["content"].strip():
                    sections.append(current_section)
                current_section = {"title": heading, "content": "", "pages": [], "tables": []}

            current_section["content"] += text + "\n"
            current_section["pages"].append(page_num)
            if tables:
                current_section["tables"].extend(tables)

        # Don't forget the last section
        if current_section["content"].strip():
            sections.append(current_section)

    log.info(f"Detected {len(sections)} sections")
    return sections


def regex_ner(text: str) -> dict:
    """Stage 2: Extract entities using regex patterns."""
    return {
        "part_numbers": list(set(RE_PART_NUMBER.findall(text))),
        "voltages": list(set(RE_VOLTAGE.findall(text))),
        "amperages": list(set(RE_AMPERAGE.findall(text))),
        "resistances": list(set(RE_RESISTANCE.findall(text))),
        "torques": list(set(RE_TORQUE.findall(text))),
        "wire_colors": list(set(RE_WIRE_COLORS.findall(text))),
        "fluids": list(set(RE_FLUID.findall(text))),
        "component_types": list(set(RE_COMPONENT_TYPES.findall(text))),
        "connectors": list(set(RE_CONNECTOR.findall(text)[:20])),  # Limit connectors (noisy)
    }


def should_skip_section(title: str, content: str) -> bool:
    """Check if section should be skipped."""
    if SKIP_PATTERNS.match(title):
        return True
    if len(content.strip()) < 200:
        return True
    # Skip if mostly page numbers or whitespace
    alpha_ratio = sum(c.isalpha() for c in content) / max(len(content), 1)
    if alpha_ratio < 0.3:
        return True
    return False


def claude_extract(client, section_title: str, section_content: str, vehicle: dict, ner_results: dict) -> Optional[dict]:
    """Stage 3: Use Claude to extract structured knowledge."""
    try:
        # Pre-populate context from NER
        ner_context = ""
        if ner_results["part_numbers"]:
            ner_context += f"\nPre-extracted part numbers: {', '.join(ner_results['part_numbers'])}"
        if ner_results["component_types"]:
            ner_context += f"\nDetected component types: {', '.join(ner_results['component_types'])}"

        response = client.messages.create(
            model="claude-sonnet-4-20250514",
            max_tokens=4096,
            system="""You are an automotive electrical engineer and knowledge graph expert.
Given a vehicle manual section, extract structured knowledge as JSON.

Output ONLY valid JSON, no markdown fences:
{
  "system": "Engine|Brakes|HVAC|Electrical|Transmission|Steering|Suspension|Body|Safety",
  "subsystem": "specific subsystem name",
  "doc_type": "procedure|wiring|specs|troubleshooting|maintenance|overview",
  "components": [
    {"name": "...", "type": "ecu|sensor|actuator|relay|fuse|connector|wire|pump|valve|motor|switch", "part_number": "...", "specs": {"voltage": "...", "amperage": "...", "resistance": "..."}}
  ],
  "relationships": [
    {"from": "component_name", "to": "component_name", "type": "CONNECTS_TO|POWERS|GROUNDS|SIGNALS|PART_OF|CONTROLS", "properties": {"wire_color": "...", "pin": "...", "protocol": "CAN|LIN|UART"}}
  ],
  "procedures": [
    {"title": "...", "steps": ["step 1", "step 2"], "tools_required": ["..."], "warnings": ["..."]}
  ],
  "maintenance": [
    {"item": "...", "interval_km": 10000, "interval_months": 6, "fluid_type": "..."}
  ],
  "confidence": 0.85
}""",
            messages=[{
                "role": "user",
                "content": f"Vehicle: {vehicle['year']} {vehicle['make']} {vehicle['model']}\n"
                           f"Manual section: {section_title}\n{ner_context}\n\n"
                           f"{section_content[:3000]}"
            }]
        )

        text = response.content[0].text.strip()
        # Strip markdown fences if present
        if text.startswith("```"):
            text = re.sub(r'^```(?:json)?\n?', '', text)
            text = re.sub(r'\n?```$', '', text)

        result = json.loads(text)

        # Calculate cost (sonnet pricing: $3/$15 per MTok)
        input_tokens = response.usage.input_tokens
        output_tokens = response.usage.output_tokens
        cost = (input_tokens * 3.0 + output_tokens * 15.0) / 1_000_000
        result["_cost_usd"] = round(cost, 6)
        result["_input_tokens"] = input_tokens
        result["_output_tokens"] = output_tokens

        return result
    except Exception as e:
        log.warning(f"Claude extraction failed for '{section_title}': {e}")
        return None


def build_output(section: dict, section_idx: int, vehicle: dict, ner: dict, claude_result: Optional[dict], pdf_path: str) -> dict:
    """Stage 4: Build JSON output matching ScrapedPost format."""
    pages = section["pages"]
    page_range = f"{pages[0]}-{pages[-1]}" if pages else ""

    system = ""
    subsystem = ""
    doc_type = "overview"
    components = []
    relationships = []
    procedures = []
    confidence = 0.5
    cost = 0.0

    if claude_result:
        system = claude_result.get("system", "")
        subsystem = claude_result.get("subsystem", "")
        doc_type = claude_result.get("doc_type", "overview")
        components = claude_result.get("components", [])
        relationships = claude_result.get("relationships", [])
        procedures = claude_result.get("procedures", [])
        confidence = claude_result.get("confidence", 0.5)
        cost = claude_result.get("_cost_usd", 0.0)

    # Build component string from NER + Claude
    comp_names = [c["name"] for c in components] if components else ner.get("component_types", [])

    make_slug = vehicle["make"].lower().replace(" ", "-")
    model_slug = vehicle["model"].lower().replace(" ", "-")
    source_id = f"manual:{make_slug}-{model_slug}-{vehicle['year']}:section-{section_idx}"

    return {
        "source": "manual",
        "source_id": source_id,
        "title": section["title"],
        "content": section["content"][:10000],  # Cap content length
        "author": "OEM",
        "url": f"file://{pdf_path}#page={pages[0] if pages else 1}",
        "published_at": f"{vehicle['year']}-01-01T00:00:00Z" if vehicle["year"] else "",
        "scraped_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "metadata": {
            "vehicle": f"{vehicle['year']} {vehicle['make']} {vehicle['model']}",
            "vehicle_info": vehicle,
            "section": f"{system}/{subsystem}" if system else section["title"],
            "components": ", ".join(comp_names[:20]),
            "doc_type": doc_type,
            "extracted_components": components,
            "extracted_relationships": relationships,
            "extracted_procedures": procedures,
            "confidence": confidence,
            "page_range": page_range,
            "part_numbers": ner["part_numbers"],
            "ner_results": {k: v for k, v in ner.items() if v},
            "claude_cost_usd": cost,
        }
    }


def process_pdf(pdf_path: str, output_dir: str, vehicle_override: Optional[dict] = None, max_pages: int = 0):
    """Main pipeline: PDF → sections → NER → Claude → JSON."""
    pdf_path = os.path.abspath(pdf_path)
    os.makedirs(output_dir, exist_ok=True)

    # Detect vehicle
    vehicle = vehicle_override or detect_vehicle_from_filename(pdf_path)
    log.info(f"Vehicle: {vehicle['year']} {vehicle['make']} {vehicle['model']}")

    # Stage 1: Extract sections
    sections = extract_sections(pdf_path, max_pages=max_pages)
    if not sections:
        log.error("No sections extracted from PDF")
        return

    # Setup Claude client (optional)
    client = None
    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if api_key:
        try:
            import anthropic
            client = anthropic.Anthropic()
            log.info("Claude API available — will use for intelligent extraction")
        except Exception as e:
            log.warning(f"Claude unavailable: {e}")
    else:
        log.info("ANTHROPIC_API_KEY not set — using regex NER only (free mode)")

    total_cost = 0.0
    total_sections = 0
    total_claude_calls = 0
    output_files = []

    make_slug = vehicle["make"].lower().replace(" ", "-")
    model_slug = vehicle["model"].lower().replace(" ", "-")

    for idx, section in enumerate(sections):
        title = section["title"]

        # Skip non-useful sections
        if should_skip_section(title, section["content"]):
            log.debug(f"  Skipping section: {title}")
            continue

        # Stage 2: Regex NER
        ner = regex_ner(section["content"])
        ner_count = sum(len(v) for v in ner.values())
        log.info(f"  Section {idx}: '{title}' ({len(section['pages'])} pages, {ner_count} NER entities)")

        # Stage 3: Claude (if available and section is substantial)
        claude_result = None
        if client and len(section["content"].strip()) > 200:
            claude_result = claude_extract(client, title, section["content"], vehicle, ner)
            if claude_result:
                total_claude_calls += 1
                cost = claude_result.get("_cost_usd", 0)
                total_cost += cost
                log.info(f"    Claude: system={claude_result.get('system','?')}, "
                        f"components={len(claude_result.get('components',[]))}, "
                        f"relationships={len(claude_result.get('relationships',[]))}, "
                        f"cost=${cost:.4f}")

        # Stage 4: Build output
        output = build_output(section, idx, vehicle, ner, claude_result, pdf_path)

        # Write JSON
        content_hash = hashlib.md5(section["content"][:500].encode()).hexdigest()[:8]
        filename = f"manual-{make_slug}-{model_slug}-{vehicle['year']}-s{idx:03d}-{content_hash}.json"
        filepath = os.path.join(output_dir, filename)

        with open(filepath, 'w') as f:
            json.dump(output, f, indent=2, default=str)

        output_files.append(filepath)
        total_sections += 1

    # Summary
    log.info(f"\n{'='*60}")
    log.info(f"Processing complete: {Path(pdf_path).name}")
    log.info(f"  Sections processed: {total_sections}/{len(sections)}")
    log.info(f"  Claude API calls:   {total_claude_calls}")
    log.info(f"  Total Claude cost:  ${total_cost:.4f}")
    log.info(f"  Output files:       {len(output_files)}")
    for f in output_files[:5]:
        log.info(f"    {f}")
    if len(output_files) > 5:
        log.info(f"    ... and {len(output_files) - 5} more")
    log.info(f"{'='*60}")


def main():
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} <pdf_path> <output_dir> [--make MAKE] [--model MODEL] [--year YEAR]")
        sys.exit(1)

    pdf_path = sys.argv[1]
    output_dir = sys.argv[2]

    if not os.path.exists(pdf_path):
        log.error(f"PDF not found: {pdf_path}")
        sys.exit(1)

    # Parse optional vehicle overrides
    vehicle = None
    max_pages = 0
    args = sys.argv[3:]
    if args:
        vehicle = detect_vehicle_from_filename(pdf_path)
        i = 0
        while i < len(args) - 1:
            if args[i] == "--make":
                vehicle["make"] = args[i + 1]; i += 2
            elif args[i] == "--model":
                vehicle["model"] = args[i + 1]; i += 2
            elif args[i] == "--year":
                vehicle["year"] = int(args[i + 1]); i += 2
            elif args[i] == "--max-pages":
                max_pages = int(args[i + 1]); i += 2
            else:
                i += 1

    process_pdf(pdf_path, output_dir, vehicle, max_pages=max_pages)


if __name__ == "__main__":
    main()
