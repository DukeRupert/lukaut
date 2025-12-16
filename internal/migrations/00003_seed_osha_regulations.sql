-- +goose Up

-- ============================================================================
-- OSHA 1926 Construction Safety Regulations Seed Data
-- ============================================================================
-- Seeds key OSHA 1926 construction standards for AI violation matching.
-- Uses ON CONFLICT for idempotent re-runs.
-- ============================================================================

-- FALL PROTECTION (Subpart M) - Most cited OSHA violation

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.501(b)(1)', 'Unprotected Sides and Edges', 'Fall Protection', 'General Requirements', 'Each employee on a walking/working surface with an unprotected side or edge 6 feet or more above a lower level shall be protected from falling by guardrail systems, safety net systems, or personal fall arrest systems. Applies to open-sided floors, platforms, runways, ramps, mezzanines, loading docks. Key hazards: unguarded edges, missing guardrails, inadequate fall protection.', 'Workers at heights of 6+ feet must have fall protection (guardrails, nets, or harness).', 'critical', '1926.501')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.501(b)(2)', 'Leading Edges', 'Fall Protection', 'General Requirements', 'Each employee constructing a leading edge 6 feet or more above lower levels shall be protected by guardrail systems, safety net systems, or personal fall arrest systems. Leading edge work includes floors, roofs, decks, formwork.', 'Workers on leading edges 6+ feet high must use fall protection.', 'critical', '1926.501')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.501(b)(4)', 'Holes', 'Fall Protection', 'General Requirements', 'Each employee on walking/working surfaces shall be protected from falling through holes including skylights more than 6 feet above lower levels by personal fall arrest systems, covers, or guardrail systems. Covers must be secured, marked HOLE or COVER, support twice expected weight.', 'Holes and skylights over 6 feet must be covered, guarded, or workers use harnesses.', 'critical', '1926.501')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.501(b)(10)', 'Roofing Work on Low-Slope Roofs', 'Fall Protection', 'Roofing', 'Employees on low-slope roofs (4:12 or less) with unprotected sides 6+ feet above lower levels need guardrails, safety nets, personal fall arrest, or warning line systems combined with other protection.', 'Low-slope roofing at 6+ feet requires fall protection systems.', 'critical', '1926.501')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.501(b)(11)', 'Steep Roofs', 'Fall Protection', 'Roofing', 'Each employee on a steep roof (greater than 4:12) with unprotected sides 6+ feet above lower levels shall use guardrails with toeboards, safety nets, or personal fall arrest systems.', 'Steep roof work (>4:12 pitch) at 6+ feet requires fall protection.', 'critical', '1926.501')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.501(b)(13)', 'Residential Construction', 'Fall Protection', 'Residential', 'Employees in residential construction 6+ feet above lower levels need guardrails, safety nets, or personal fall arrest. Exception with documented fall protection plan if conventional methods infeasible.', 'Residential framing/roofing at 6+ feet needs fall protection or documented plan.', 'critical', '1926.501')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.502(b)', 'Guardrail Systems Criteria', 'Fall Protection', 'Guardrails', 'Top rails must be 42 inches (+/- 3 inches) high. Midrails required midway between top rail and floor. Must withstand 200 lbs force. No openings allowing 19-inch sphere passage. Smooth surfaces, no sharp edges.', 'Guardrails: 42 inches high, midrails required, must withstand 200 lbs.', 'serious', '1926.502')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.502(d)', 'Personal Fall Arrest Systems', 'Fall Protection', 'Fall Arrest', 'Anchorages must support 5,000 lbs per worker or have 2x safety factor. Full body harnesses required (no body belts). D-ring between shoulder blades. Lifelines protected from cuts/abrasion. Inspect before each use.', 'Fall arrest: 5,000 lb anchors, full harnesses only, inspect before use.', 'critical', '1926.502')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.502(i)', 'Covers', 'Fall Protection', 'Covers', 'Covers for holes must support twice the weight of employees, equipment, materials. Must be secured to prevent displacement, color coded or marked HOLE or COVER. Roadway covers must support 2x max vehicle axle load.', 'Hole covers must support 2x expected load and be secured/marked.', 'serious', '1926.502')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.503(a)', 'Fall Protection Training', 'Fall Protection', 'Training', 'Employers must train employees exposed to fall hazards to recognize hazards and use fall protection equipment properly. Training before work begins. Covers hazard recognition, equipment use, procedures.', 'Workers exposed to fall hazards must receive fall protection training.', 'serious', '1926.503')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

-- SCAFFOLDING (Subpart L) - Second most cited

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.451(a)', 'Scaffold Capacity', 'Scaffolding', 'Capacity', 'Scaffolds must support 4x maximum intended load without failure. Suspension scaffold rigging must support 6x intended load. Do not exceed rated capacity. Platform deflection max 1/60 of span.', 'Scaffolds must support 4x intended load (6x for suspension).', 'critical', '1926.451')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.451(b)', 'Scaffold Platform Construction', 'Scaffolding', 'Platforms', 'Platforms must be fully planked/decked, at least 18 inches wide. Front edge within 14 inches of work face. Gaps max 1 inch. Planks overlap 12 inches minimum or extend 6 inches past support. Secured to prevent movement.', 'Scaffold platforms: fully planked, 18 inches wide, within 14 inches of work.', 'serious', '1926.451')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.451(c)', 'Supported Scaffold Requirements', 'Scaffolding', 'Supported Scaffolds', 'Scaffolds with height to base ratio over 4:1 must be tied, guyed, or braced. Legs must bear on base plates and mudsills on firm foundation. Level footings required. Competent person supervision for erection.', 'Tall scaffolds (>4:1 ratio) must be tied/braced; firm footing required.', 'critical', '1926.451')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.451(e)', 'Scaffold Access', 'Scaffolding', 'Access', 'When platforms are more than 2 feet above/below access point, provide ladders, stair towers, ramps, or integral access. Cross-braces cannot be used for climbing. Safe access required at all working levels.', 'Scaffolds over 2 feet need proper access (ladders/stairs), not cross-braces.', 'serious', '1926.451')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.451(f)', 'Scaffold Use Requirements', 'Scaffolding', 'Use', 'Do not move scaffolds horizontally with workers on them unless designed for such movement. No use during storms/high winds. Keep platforms clear of debris. Competent person must inspect before each shift.', 'Do not move scaffolds with workers; inspect before each shift.', 'serious', '1926.451')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.451(g)', 'Scaffold Fall Protection', 'Scaffolding', 'Fall Protection', 'Workers on scaffolds more than 10 feet above lower level need guardrails or personal fall arrest. Guardrails: 42 inches high with midrails and toeboards. Cross-bracing acceptable as midrail if meets height requirements.', 'Workers on scaffolds 10+ feet high need guardrails or fall arrest.', 'critical', '1926.451')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.451(h)', 'Falling Object Protection', 'Scaffolding', 'Falling Objects', 'Toeboards required where workers below could be struck. Minimum 3.5 inches high, withstand 50 lbs upward force. Provide canopies, screens, or debris nets for overhead hazards. Barricade areas below scaffold.', 'Toeboards (3.5 inch min) required where objects could fall on workers below.', 'serious', '1926.451')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.452(w)', 'Mobile Scaffolds', 'Scaffolding', 'Mobile Scaffolds', 'Mobile scaffolds must sustain 4x intended load. Caster stems pinned/secured in legs. Wheels locked during use. Workers only ride if level surface, no holes/obstructions, manual movement force, within floor load limits.', 'Mobile scaffolds: locked casters; workers ride only under safe conditions.', 'serious', '1926.452')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.453(b)', 'Aerial Lift Requirements', 'Scaffolding', 'Aerial Lifts', 'Test lift controls daily before use. Only authorized trained operators. Body belt/harness with lanyard attached to boom or basket required. Do not belt off to adjacent structures. Maintain safe distance from power lines.', 'Aerial lifts: daily tests, trained operators, fall protection attached to boom.', 'serious', '1926.453')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.454(a)', 'Scaffold Training', 'Scaffolding', 'Training', 'Train workers on scaffold hazards, proper erection/disassembly, fall protection use, electrical hazards, load limits. Competent person must train. Retrain when workplace changes or deficiencies observed.', 'Scaffold workers must be trained on hazards, procedures, and fall protection.', 'serious', '1926.454')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

-- LADDERS (Subpart X)

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.1053(b)(1)', 'Ladder Positioning', 'Ladders', 'Use Requirements', 'Non-self-supporting ladders positioned at 4:1 angle (75.5 degrees). Base 1 foot out for every 4 feet of height. Ladder must not slip at top or bottom. Secure or have worker hold base.', 'Ladders at 4:1 angle: 1 foot out for every 4 feet up.', 'serious', '1926.1053')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.1053(b)(4)', 'Ladder Extension Above Landing', 'Ladders', 'Use Requirements', 'Ladders used to access upper landing must extend at least 3 feet above landing surface or have grab rails. Provides secure handhold for mounting/dismounting. Side rails extend above landing.', 'Ladders must extend 3 feet above landing for safe access.', 'serious', '1926.1053')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.1053(b)(5)', 'Secured Ladders', 'Ladders', 'Use Requirements', 'Ladders must be secured to prevent displacement. Tie off at top, secure base, or have worker hold. Single cleat ladders over 24 feet need landing platforms every 12 feet.', 'Secure ladders to prevent slipping or displacement.', 'serious', '1926.1053')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.1053(b)(6)', 'Defective Ladders', 'Ladders', 'Inspection', 'Remove defective ladders from service immediately. Tag or mark as dangerous. Do not repair metal ladders. Defects include broken rungs, split side rails, corrosion, missing hardware.', 'Remove defective ladders from service; do not use or repair.', 'serious', '1926.1053')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.1053(b)(13)', 'Top Step Prohibition', 'Ladders', 'Use Requirements', 'Do not stand on top step or top cap of stepladder. Top two rungs of extension ladder not for standing. Maintain three points of contact while climbing.', 'Do not stand on top step of stepladder or top rungs of extension ladder.', 'serious', '1926.1053')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.1053(b)(16)', 'Ladder Load Capacity', 'Ladders', 'Capacity', 'Ladders must support intended load. Do not exceed maximum weight rating. Consider worker weight plus tools and materials. Check duty rating label before use.', 'Do not exceed ladder weight rating; check duty rating.', 'serious', '1926.1053')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.1052(a)', 'Stairway Requirements', 'Ladders', 'Stairways', 'Stairways with 4+ risers or rising 30+ inches need handrails. Stair rails 30-37 inches from stair tread surface. Stairway width minimum 22 inches. Rise/run consistent throughout.', 'Stairways need handrails if 4+ risers or 30+ inches high.', 'serious', '1926.1052')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.1060(a)', 'Ladder Training', 'Ladders', 'Training', 'Train employees on ladder hazards, proper use, load capacities, 4:1 positioning, three-point contact. Competent person training required. Retrain when deficiencies observed.', 'Train workers on ladder hazards, positioning, and safe use.', 'serious', '1926.1060')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

-- EXCAVATIONS (Subpart P)

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.651(c)', 'Excavation Access and Egress', 'Excavations', 'Access', 'Excavations 4+ feet deep must have ladders, steps, ramps, or other safe access within 25 feet of workers. Structural ramps used for access must be designed by competent person.', 'Excavations 4+ feet need access (ladder/ramp) within 25 feet.', 'critical', '1926.651')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.651(j)', 'Protection from Loose Rock', 'Excavations', 'Hazards', 'Protect workers from loose rock or soil that could fall into excavation. Scale or remove loose material. Install protective barricades. Keep spoil pile minimum 2 feet from edge.', 'Protect from falling rock/soil; keep spoils 2+ feet from excavation edge.', 'serious', '1926.651')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.651(k)', 'Excavation Inspections', 'Excavations', 'Inspections', 'Competent person must inspect excavations daily and after rainstorms, vibration, or other hazard-increasing events. Remove workers if hazardous conditions found. Inspect adjacent areas.', 'Competent person inspects excavations daily and after weather/vibration.', 'serious', '1926.651')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.652(a)', 'Cave-in Protection Required', 'Excavations', 'Protective Systems', 'Excavations 5+ feet deep require cave-in protection unless excavation is in stable rock. Use sloping, benching, shoring, or shielding. Competent person determines soil type and protective system.', 'Excavations 5+ feet need cave-in protection (sloping/shoring/shields).', 'critical', '1926.652')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.652(b)', 'Sloping Requirements', 'Excavations', 'Protective Systems', 'Slope excavation walls based on soil classification. Type A: 3/4:1 (53 degrees). Type B: 1:1 (45 degrees). Type C: 1.5:1 (34 degrees). Competent person classifies soil. Maximum slope may not exceed these angles.', 'Slope excavations per soil type: A=3/4:1, B=1:1, C=1.5:1.', 'critical', '1926.652')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.651(b)', 'Underground Installations', 'Excavations', 'Utilities', 'Contact utility companies before digging to locate underground installations. Hand dig within tolerance zone of marked utilities. Protect, support, or remove underground installations as necessary.', 'Locate utilities before digging; hand dig near marked lines.', 'critical', '1926.651')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

-- PERSONAL PROTECTIVE EQUIPMENT (Subpart E)

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.100(a)', 'Hard Hat Requirements', 'Personal Protective Equipment', 'Head Protection', 'Employees working where there is danger of head injury from impact, falling or flying objects, or electrical shock shall wear protective helmets. Hard hats required for overhead work, falling objects, struck-by hazards. Safety helmets must meet ANSI standards.', 'Hard hats required where head injury hazards exist (falling objects, impacts).', 'serious', '1926.100')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.102(a)', 'Eye and Face Protection', 'Personal Protective Equipment', 'Eye Protection', 'Eye and face protection required when exposed to flying particles, molten metal, liquid chemicals, acids, caustic liquids, chemical gases, vapors, or potentially injurious light radiation. Safety glasses, goggles, face shields as appropriate.', 'Eye protection required for flying particles, chemicals, welding.', 'serious', '1926.102')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.101(a)', 'Hearing Protection', 'Personal Protective Equipment', 'Hearing Protection', 'Feasible engineering or administrative controls must be used when workers exposed to noise exceeding limits. If controls insufficient, provide hearing protection. Earplugs, earmuffs required in high-noise areas.', 'Hearing protection required when noise exceeds exposure limits.', 'serious', '1926.101')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.95(a)', 'PPE Assessment', 'Personal Protective Equipment', 'General', 'Employer must assess workplace to determine if hazards requiring PPE are present. Hazard assessment documented. Select PPE that properly fits each worker. Defective or damaged PPE must not be used.', 'Assess workplace hazards and provide appropriate PPE to workers.', 'serious', '1926.95')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.28(a)', 'Personal Protective Equipment General', 'Personal Protective Equipment', 'General', 'Employer responsible for requiring use of appropriate personal protective equipment in all operations where exposure to hazardous conditions exists. Includes head, eye, face, hand, foot protection.', 'Employers must require appropriate PPE for hazardous conditions.', 'serious', '1926.28')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.96', 'Foot Protection', 'Personal Protective Equipment', 'Foot Protection', 'Safety-toe footwear required for employees exposed to foot injuries from falling or rolling objects, objects piercing sole, or electrical hazards. Steel-toe boots, metatarsal guards as appropriate.', 'Safety-toe boots required where foot crush or puncture hazards exist.', 'serious', '1926.96')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.103', 'Respiratory Protection', 'Personal Protective Equipment', 'Respiratory', 'Respirators required when engineering controls not feasible or during installation of controls. Must comply with 29 CFR 1910.134. Written respiratory protection program required. Fit testing, training, medical evaluation needed.', 'Respirators required when engineering controls insufficient; fit testing required.', 'serious', '1926.103')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.95(d)', 'High Visibility Apparel', 'Personal Protective Equipment', 'Visibility', 'High-visibility safety apparel required for workers exposed to public vehicular traffic. Meets ANSI/ISEA 107 Class 2 or 3 standards. Reflective vests, shirts, or jackets with retroreflective striping.', 'High-visibility apparel required near public traffic.', 'serious', '1926.95')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

-- ELECTRICAL SAFETY (Subpart K)

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.405(a)(2)(ii)', 'GFCI Protection', 'Electrical Safety', 'Wiring', 'Ground-fault circuit interrupters (GFCIs) required for all 120-volt, single-phase, 15- and 20-ampere receptacles on construction sites not part of permanent wiring. Alternative: assured equipment grounding conductor program.', 'GFCIs required for temporary 120V receptacles on construction sites.', 'critical', '1926.405')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.404(b)', 'Grounding Requirements', 'Electrical Safety', 'Grounding', 'Exposed non-current-carrying metal parts of equipment must be grounded. Ground path must be permanent and continuous. Grounding conductor cannot be smaller than largest ungrounded conductor.', 'Equipment metal parts must be properly grounded.', 'critical', '1926.404')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.416(a)', 'Protection from Electrical Hazards', 'Electrical Safety', 'Work Practices', 'No employee shall work near any part of an electric power circuit unless protected against electric shock by de-energizing, guarding, or insulating. Minimum clearance distances from power lines required.', 'Protect workers from electrical shock; maintain clearance from power lines.', 'critical', '1926.416')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.417', 'Lockout and Tagging', 'Electrical Safety', 'Lockout/Tagout', 'Controls must be locked out and tagged to prevent accidental energization during maintenance. Only qualified persons may work on energized circuits. Tags must warn against operation.', 'Lock out and tag electrical controls during maintenance.', 'critical', '1926.417')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.405(g)', 'Flexible Cords and Cables', 'Electrical Safety', 'Wiring', 'Flexible cords must be rated for hard or extra-hard usage. No splices except with proper connectors. Cords cannot run through holes, doorways, or windows where damaged. Protect from physical damage.', 'Use hard-usage rated cords; no improper splices; protect from damage.', 'serious', '1926.405')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.405(j)', 'Temporary Wiring', 'Electrical Safety', 'Wiring', 'Temporary electrical power and lighting installations must meet specific requirements. Receptacles must have covers. Branch circuits not to exceed 20 amperes. Lamp protection required. Remove temporary wiring promptly when no longer needed.', 'Temporary wiring must have covers, proper circuits, lamp guards.', 'serious', '1926.405')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.416(e)', 'Portable Equipment Handling', 'Electrical Safety', 'Equipment', 'Worn or frayed electric cords must not be used. Extension cords must be 3-wire type. Attachment plugs and receptacles cannot be connected or altered to allow mismatched plugs. Inspect cords before use.', 'No worn/frayed cords; use 3-wire extensions; inspect before use.', 'serious', '1926.416')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.431', 'Electrical Equipment Maintenance', 'Electrical Safety', 'Maintenance', 'Electrical equipment must be maintained in safe condition free from hazards. Listed or labeled equipment used according to instructions. Damaged equipment removed from service.', 'Maintain electrical equipment safely; remove damaged equipment.', 'serious', '1926.431')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

-- HOUSEKEEPING

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.25(a)', 'Debris Removal', 'Housekeeping', 'General', 'During construction, alteration, or repairs, form and scrap lumber with protruding nails and other debris shall be kept clear from work areas, passageways, and stairs. Nails must be pulled from lumber before stacking.', 'Keep work areas clear of debris; remove nails from scrap lumber.', 'other', '1926.25')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.25(b)', 'Combustible Scrap', 'Housekeeping', 'Fire Prevention', 'Combustible scrap and debris must be removed at regular intervals during construction. Adequate firefighting equipment required during construction. Fire watch required during hot work.', 'Remove combustible debris regularly; maintain fire equipment.', 'serious', '1926.25')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.25(c)', 'Waste Containers', 'Housekeeping', 'General', 'Containers must be provided for collection and separation of waste, trash, oily rags, and similar combustible materials. Keep aisles and passageways clear. Proper disposal of hazardous waste required.', 'Provide containers for waste; keep passageways clear.', 'other', '1926.25')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.20(b)', 'Accident Prevention Programs', 'Housekeeping', 'Safety Programs', 'Employers must implement accident prevention programs as required by the employer''s State plan. Frequent and regular inspections of job sites, materials, and equipment by competent persons.', 'Implement accident prevention programs; conduct regular inspections.', 'other', '1926.20')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

-- HEAVY EQUIPMENT (Subpart O)

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.602(a)(9)', 'ROPS Requirements', 'Heavy Equipment', 'Equipment', 'Rollover protective structures (ROPS) required on rubber-tired, self-propelled scrapers, front-end loaders, dozers, and similar equipment. ROPS must meet minimum performance criteria. Seat belts required with ROPS.', 'ROPS and seat belts required on scrapers, loaders, dozers.', 'critical', '1926.602')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.601(b)(4)', 'Vehicle Seat Belts', 'Heavy Equipment', 'Vehicles', 'Seat belts must be provided on all motor vehicles and worn by all occupants. Seat belt assemblies must meet specifications. Employer must require use of seat belts by all motor vehicle operators.', 'Seat belts required on all motor vehicles; must be worn.', 'serious', '1926.601')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.601(b)(6)', 'Backing Vehicles', 'Heavy Equipment', 'Vehicles', 'Vehicles with obstructed rear view must have reverse signal alarm audible above surrounding noise or use observer when backing. Backup alarms, cameras, or spotters required. Train operators on backing hazards.', 'Use backup alarms or spotters when rear view obstructed.', 'serious', '1926.601')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.600(a)', 'Equipment Safety Requirements', 'Heavy Equipment', 'Equipment', 'All equipment left unattended at night must have lights, reflectors, or barricades for warning. Parked equipment must have parking brake engaged. No riders except in authorized seat positions.', 'Unattended equipment needs lights/reflectors; engage parking brakes.', 'serious', '1926.600')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.602(c)', 'Lifting and Hauling Equipment', 'Heavy Equipment', 'Equipment', 'Industrial trucks must meet ASME B56.1 requirements. Modifications and additions affecting capacity and safe operation prohibited without manufacturer approval. Operators must be trained and competent.', 'Industrial trucks meet ASME standards; trained operators only.', 'serious', '1926.602')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

-- MATERIAL HANDLING (Subpart H)

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.250(a)', 'General Material Storage', 'Material Handling', 'Storage', 'All materials stored in tiers shall be stacked, blocked, interlocked, or secured to prevent sliding, falling, or collapse. Storage areas free from hazards. Aisles and passageways kept clear.', 'Stack stored materials securely; keep aisles clear.', 'serious', '1926.250')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.250(a)(2)', 'Storage Height Limits', 'Material Handling', 'Storage', 'Lumber stacked max 16 feet if manually handled, 20 feet with mechanical equipment. Brick stacks max 7 feet. Block stacks max 6 feet. Used lumber must have nails removed before stacking.', 'Height limits: lumber 16-20 ft, brick 7 ft, block 6 ft.', 'serious', '1926.250')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.251(a)', 'Rigging Equipment Inspection', 'Material Handling', 'Rigging', 'Rigging equipment for material handling shall be inspected before use each shift. Defective rigging removed from service. Includes slings, chains, ropes, hooks, shackles. Document inspections.', 'Inspect rigging before each shift; remove defective equipment.', 'serious', '1926.251')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.251(c)', 'Wire Rope Inspection', 'Material Handling', 'Rigging', 'Remove wire rope from service when six randomly distributed broken wires in one rope lay, or three broken wires in one strand in one rope lay. Also remove for kinking, crushing, bird caging, heat damage, corrosion.', 'Remove wire rope with 6+ broken wires per lay or visible damage.', 'serious', '1926.251')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES ('1926.251(e)', 'Synthetic Web Slings', 'Material Handling', 'Rigging', 'Synthetic web slings must have identification tag showing rated capacities. Remove from service if acid/caustic burns, melting, charring, snags, cuts, or broken stitches. Repairs by manufacturer only.', 'Synthetic slings need capacity tags; remove if burned, cut, or damaged.', 'serious', '1926.251')
ON CONFLICT (standard_number) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, full_text = EXCLUDED.full_text, summary = EXCLUDED.summary, severity_typical = EXCLUDED.severity_typical, updated_at = NOW();

-- +goose Down
DELETE FROM regulations WHERE standard_number LIKE '1926.%';
