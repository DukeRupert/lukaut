-- +goose Up

-- ============================================================================
-- OSHA 1926 Construction Safety Regulations Seed Data
-- ============================================================================
-- This migration seeds ~175 key OSHA 1926 construction standards optimized
-- for full-text search matching with AI-detected violations.
-- ============================================================================

-- FALL PROTECTION (Subpart M)
-- Most cited OSHA violation category
INSERT INTO regulations (standard_number, title, category, subcategory, full_text, summary, severity_typical, parent_standard)
VALUES
(
    '1926.501(b)(1)',
    'Unprotected Sides and Edges',
    'Fall Protection',
    'General Requirements',
    'Each employee on a walking/working surface with an unprotected side or edge 6 feet or more above a lower level shall be protected from falling by guardrail systems, safety net systems, or personal fall arrest systems. Applies to open-sided floors, platforms, runways, ramps, mezzanines, loading docks. Key hazards: unguarded edges, missing guardrails, inadequate fall protection, exposed edges, open sides.',
    'Workers at heights of 6+ feet must have fall protection (guardrails, nets, or harness).',
    'critical',
    '1926.501'
),
(
    '1926.501(b)(2)',
    'Leading Edges',
    'Fall Protection',
    'General Requirements',
    'Each employee who is constructing a leading edge 6 feet or more above lower levels shall be protected from falling by guardrail systems, safety net systems, or personal fall arrest systems. Leading edge work includes construction of floors, roofs, decks, formwork. Exception: when employer demonstrates guardrails or nets are infeasible.',
    'Workers on leading edges 6+ feet high must use fall protection or prove infeasibility.',
    'critical',
    '1926.501'
),
(
    '1926.501(b)(3)',
    'Hoist Areas',
    'Fall Protection',
    'General Requirements',
    'Each employee in a hoist area shall be protected from falling 6 feet or more by guardrail systems or personal fall arrest systems. If guardrail system or portion removed to facilitate hoisting, chain or gate must be installed. Hoist areas require fall protection barriers, safety harness, or personal fall arrest system.',
    'Workers in hoist areas 6+ feet high need guardrails or personal fall arrest.',
    'critical',
    '1926.501'
),
(
    '1926.501(b)(4)',
    'Holes',
    'Fall Protection',
    'General Requirements',
    'Each employee on walking/working surfaces shall be protected from falling through holes (including skylights) more than 6 feet above lower levels by personal fall arrest systems, covers, or guardrail systems. Covers must be secured to prevent displacement, color coded or marked "HOLE" or "COVER", able to support twice the weight of employees, equipment, materials. Includes floor holes, roof holes, wall openings, skylights.',
    'Holes and skylights over 6 feet must be covered, guarded, or workers use harnesses.',
    'critical',
    '1926.501'
),
(
    '1926.501(b)(5)',
    'Formwork and Reinforcing Steel',
    'Fall Protection',
    'General Requirements',
    'Each employee on the face of formwork or reinforcing steel shall be protected from falling 6 feet or more by personal fall arrest systems, safety net systems, or positioning device systems. Requirements apply to wall forming, vertical concrete forming, rebar installation, reinforcing steel assembly. Fall arrest harness, positioning devices, safety nets required.',
    'Workers on formwork or rebar 6+ feet high must use fall arrest or positioning devices.',
    'critical',
    '1926.501'
),
(
    '1926.501(b)(7)',
    'Holes in Residential Construction',
    'Fall Protection',
    'Residential Construction',
    'Each employee engaged in residential construction activities 6 feet or more above lower levels shall be protected by guardrail systems with toeboards, safety net systems, or personal fall arrest systems. Includes work on roofs, floors, elevated surfaces in residential construction. Applies to single-family homes, townhouses, condos up to three stories.',
    'Residential construction workers at 6+ feet need fall protection.',
    'critical',
    '1926.501'
),
(
    '1926.501(b)(10)',
    'Roofing Work on Low-Slope Roofs',
    'Fall Protection',
    'Roofing',
    'Each employee engaged in roofing activities on low-slope roofs with unprotected sides and edges 6 feet or more above lower levels shall be protected from falling by guardrail systems, safety net systems, personal fall arrest systems, or a combination of warning line system and guardrail system, warning line system and safety net system, warning line system and personal fall arrest system, or warning line system and safety monitoring system. Low-slope roofs have slopes less than or equal to 4:12.',
    'Low-slope roofing work at 6+ feet requires fall protection systems.',
    'critical',
    '1926.501'
),
(
    '1926.501(b)(11)',
    'Steep Roofs',
    'Fall Protection',
    'Roofing',
    'Each employee on a steep roof with unprotected sides and edges 6 feet or more above lower levels shall be protected from falling by guardrail systems with toeboards, safety net systems, or personal fall arrest systems. Steep roofs have slopes greater than 4:12 (rise/run).',
    'Steep roof work (>4:12 pitch) at 6+ feet requires fall protection.',
    'critical',
    '1926.501'
),
(
    '1926.501(b)(13)',
    'Residential Construction',
    'Fall Protection',
    'Residential Construction',
    'Each employee engaged in residential construction activities 6 feet or more above lower levels shall be protected by guardrail systems, safety net systems, or personal fall arrest systems. Exception: When employer can demonstrate infeasibility, may implement fall protection plan meeting 1926.502(k). Applies to framing, roofing, exterior wall work.',
    'Residential framing/roofing at 6+ feet needs fall protection or documented plan.',
    'critical',
    '1926.501'
),
(
    '1926.502(b)(1)',
    'Guardrail System Top Rail Height',
    'Fall Protection',
    'Guardrail Systems',
    'Top edge height of top rails must be 42 inches plus or minus 3 inches above the walking/working level. When conditions warrant, height may exceed 45 inches provided system meets all other criteria. Guardrails must be installed along all open sides, edges of platforms, walkways, runways. Toprail height critical for fall prevention.',
    'Guardrail top rails must be 39-45 inches high (42 inches standard).',
    'serious',
    '1926.502'
),
(
    '1926.502(b)(2)',
    'Guardrail Midrails',
    'Fall Protection',
    'Guardrail Systems',
    'Midrails, screens, mesh, intermediate vertical members, or equivalent intermediate structural members shall be installed between the top edge of the guardrail system and the walking/working surface when there is no wall or parapet wall at least 21 inches high. Midrails must be midway between top edge of guardrail and walking/working level. Prevents workers from falling through guardrail opening.',
    'Guardrails need midrails or screens between top rail and floor.',
    'serious',
    '1926.502'
),
(
    '1926.502(b)(3)',
    'Guardrail Strength Requirements',
    'Fall Protection',
    'Guardrail Systems',
    'Guardrail systems shall be capable of withstanding, without failure, a force of at least 200 pounds applied within 2 inches of the top edge, in any outward or downward direction, at any point along the top edge. System must prevent workers from falling over top rail or through openings. Load testing required to verify strength.',
    'Guardrails must withstand 200 lbs force applied to top rail.',
    'serious',
    '1926.502'
),
(
    '1926.502(b)(10)',
    'Guardrail Gaps and Openings',
    'Fall Protection',
    'Guardrail Systems',
    'Guardrail systems shall be surfaced to prevent injury such as punctures or lacerations and to prevent snagging of clothing. Top rails and midrails shall not have rough edges, splinters, sharp points. Openings must not allow passage of 19-inch diameter sphere. Prevents workers from slipping through gaps.',
    'Guardrail openings cannot exceed 19 inches; no sharp edges allowed.',
    'serious',
    '1926.502'
),
(
    '1926.502(c)(1)',
    'Safety Net Systems',
    'Fall Protection',
    'Safety Net Systems',
    'Safety nets shall be installed as close as practicable under the walking/working surface on which employees are working, but in no case more than 30 feet below such level. When nets are used on bridges, minimum required vertical distance may be greater than 30 feet. Nets extend outward from outermost projection of work surface based on distance of net below work.',
    'Safety nets must be within 30 feet below work surface.',
    'serious',
    '1926.502'
),
(
    '1926.502(c)(3)',
    'Safety Net Drop Testing',
    'Fall Protection',
    'Safety Net Systems',
    'Safety nets shall be drop-tested at the jobsite after initial installation and before being used as a fall protection system, whenever relocated, after major repair, and at 6-month intervals if left in one place. Drop-test consists of a 400-pound bag of sand 30 inches in diameter dropped into the net from highest walking/working surface.',
    'Safety nets must be drop-tested after install, repairs, moves, and every 6 months.',
    'serious',
    '1926.502'
),
(
    '1926.502(d)(1)',
    'Personal Fall Arrest System Anchorage',
    'Fall Protection',
    'Personal Fall Arrest Systems',
    'Anchorages used for attachment of personal fall arrest equipment shall be independent of any anchorage being used to support or suspend platforms and capable of supporting at least 5,000 pounds per employee attached, or shall be designed, installed, and used under supervision of qualified person as part of complete personal fall arrest system which maintains safety factor of at least two. Structural anchors, roof anchors, beam anchors must be properly rated.',
    'Fall arrest anchors must support 5,000 lbs per worker or have 2x safety factor.',
    'critical',
    '1926.502'
),
(
    '1926.502(d)(15)',
    'Personal Fall Arrest Anchorage',
    'Fall Protection',
    'Personal Fall Arrest Systems',
    'Anchorages used for attachment of personal fall arrest equipment shall be independent of any anchorage being used to support or suspend platforms. Anchorages shall be designed to support loads specified in 1926.502(d)(16). Prevents overloading single anchor point with multiple systems.',
    'Fall arrest anchors must be separate from platform support anchors.',
    'critical',
    '1926.502'
),
(
    '1926.502(d)(16)',
    'Personal Fall Arrest System Components',
    'Fall Protection',
    'Personal Fall Arrest Systems',
    'Body belts shall not be used as part of a personal fall arrest system. Full body harnesses are required. Body harness must be worn with D-ring positioned between shoulder blades. Lifelines must be protected against cuts, abrasion, melting. Connectors must be drop forged, pressed, formed steel or equivalent materials with corrosion-resistant finish.',
    'Only full-body harnesses allowed for fall arrest, no body belts.',
    'critical',
    '1926.502'
),
(
    '1926.502(d)(20)',
    'Personal Fall Arrest Inspection',
    'Fall Protection',
    'Personal Fall Arrest Systems',
    'Personal fall arrest systems and components shall be inspected prior to each use for wear, damage, and other deterioration. Defective components shall be removed from service. Inspection includes checking harness webbing, D-rings, buckles, lanyards, lifelines, shock absorbers, connectors, anchorages for cuts, fraying, burns, corrosion, deformation.',
    'Fall arrest equipment must be inspected before each use; remove damaged gear.',
    'serious',
    '1926.502'
),
(
    '1926.502(e)',
    'Positioning Device Systems',
    'Fall Protection',
    'Positioning Devices',
    'Positioning device systems shall be rigged such that an employee cannot free fall more than 2 feet. Anchorages used for positioning devices shall be capable of supporting at least twice the potential impact load of an employee''s fall or 3,000 pounds, whichever is greater. Used by workers on poles, towers, vertical surfaces. Includes body belts, positioning lanyards.',
    'Positioning devices must limit free fall to 2 feet and support 3,000 lbs.',
    'serious',
    '1926.502'
),
(
    '1926.502(f)',
    'Warning Line Systems',
    'Fall Protection',
    'Warning Lines',
    'Warning lines shall consist of ropes, wires, or chains and supporting stanchions erected to warn employees that they are approaching an unprotected roof side or edge. Must be rigged and supported so lowest point is no less than 34 inches from walking/working surface and highest point is no more than 39 inches. Set up 6 feet from roof edge. Flagged at 6-foot intervals with high-visibility material.',
    'Warning lines mark 6-foot setback from roof edges, 34-39 inches high.',
    'other',
    '1926.502'
),
(
    '1926.502(g)',
    'Controlled Access Zones',
    'Fall Protection',
    'Controlled Access Zones',
    'Controlled access zones shall be defined by a control line or other means that restricts access. When used for leading edge work, control line shall extend from edge a minimum distance of 6 feet. When used for overhand bricklaying, extend minimum of 10 feet. Only authorized employees permitted in zone. Used in combination with other fall protection.',
    'Controlled access zones limit worker entry to fall hazard areas (6-10 foot setback).',
    'other',
    '1926.502'
),
(
    '1926.502(h)',
    'Safety Monitoring Systems',
    'Fall Protection',
    'Safety Monitoring',
    'Employer shall designate competent person to monitor safety of employees and warn them when unsafe. Monitor must be on same walking/working surface and within visual sighting distance of employees. Safety monitor has no other responsibilities which could take attention from monitoring function. Used for low-slope roofing, leading edge work when other methods infeasible.',
    'Safety monitors must watch workers continuously with no other duties.',
    'other',
    '1926.502'
),
(
    '1926.502(i)',
    'Covers for Holes',
    'Fall Protection',
    'Covers',
    'Covers located in roadways and vehicular aisles shall be capable of supporting at least twice the maximum axle load of the largest vehicle to which cover might be subjected. All other covers must support twice the weight of employees, equipment, and materials that may be imposed on cover. Covers must be secured to prevent accidental displacement, color coded or marked "HOLE" or "COVER".',
    'Hole covers must support 2x expected load and be secured/marked.',
    'serious',
    '1926.502'
),
(
    '1926.502(j)',
    'Fall Protection Plan',
    'Fall Protection',
    'Fall Protection Plans',
    'Fall protection plan must be prepared by qualified person and developed specifically for site where leading edge work, precast concrete, or residential construction work will take place. Plan must document reasons why conventional fall protection is infeasible or creates greater hazard, describe alternative fall protection measures being used, identify work areas where plan applies.',
    'Fall protection plans required when conventional methods are infeasible.',
    'other',
    '1926.502'
),
(
    '1926.503(a)(1)',
    'Fall Protection Training Requirements',
    'Fall Protection',
    'Training',
    'Employer shall provide training for each employee who might be exposed to fall hazards. Program shall enable each employee to recognize fall hazards and train employees in procedures to minimize hazards. Training required before employee begins work requiring fall protection. Training must cover nature of fall hazards, correct procedures for erecting, maintaining, disassembling fall protection systems, proper use of equipment.',
    'Workers exposed to fall hazards must receive fall protection training.',
    'serious',
    '1926.503'
),
(
    '1926.503(a)(2)',
    'Fall Protection Retraining',
    'Fall Protection',
    'Training',
    'Retraining must be provided when employer has reason to believe employee does not have understanding or skill required. Situations include changes in workplace rendering previous training obsolete, changes in types of fall protection systems, inadequacies in employee knowledge or use of fall protection.',
    'Fall protection retraining required when workplace changes or deficiencies identified.',
    'serious',
    '1926.503'
),
(
    '1926.503(b)',
    'Certification of Training',
    'Fall Protection',
    'Training',
    'Employer shall verify compliance with training requirements by preparing written certification record. Certification shall contain name of employee trained, date of training, signature of person who conducted training. Latest training certification shall be maintained.',
    'Fall protection training must be documented with employee name, date, trainer signature.',
    'other',
    '1926.503'
),

-- SCAFFOLDING (Subpart L)
-- Second most cited OSHA violation
(
    '1926.451(a)(1)',
    'Scaffold Load Capacity',
    'Scaffolding',
    'Capacity',
    'Each scaffold and scaffold component shall be capable of supporting, without failure, its own weight and at least 4 times the maximum intended load applied or transmitted to it. Suspension scaffold rigging must support 6 times intended load. Load capacity must be determined by qualified person.',
    'Scaffolds must support 4x intended load (6x for suspension scaffolds).',
    'critical',
    '1926.451'
),
(
    '1926.451(a)(3)',
    'Scaffold Platform Load Capacity',
    'Scaffolding',
    'Capacity',
    'Scaffolds shall not be loaded in excess of their maximum intended loads or rated capacities, whichever is less. Platform must not deflect more than 1/60 of span when loaded. Includes weight of workers, materials, equipment. Overloading causes structural failure, collapse.',
    'Do not exceed scaffold rated capacity or load platforms beyond limits.',
    'critical',
    '1926.451'
),
(
    '1926.451(b)(1)',
    'Scaffold Platform Construction',
    'Scaffolding',
    'Platforms',
    'Each platform on all working levels of scaffolds shall be fully planked or decked. Platforms must be at least 18 inches wide. Front edge of platform cannot be more than 14 inches from face of work, except for outrigger scaffolds and plastering/lathing operations. Gaps between planks cannot exceed 1 inch. Platforms must be secured to prevent movement.',
    'Scaffold platforms must be fully planked, at least 18 inches wide, within 14 inches of work.',
    'serious',
    '1926.451'
),
(
    '1926.451(b)(3)',
    'Scaffold Platform Overlap',
    'Scaffolding',
    'Platforms',
    'Platforms shall not deflect more than 1/60 of the span when loaded. Planks must overlap minimum 12 inches or extend over centerline support at least 6 inches. Plank ends must be secured to prevent sliding off supports. Platform planks must be scaffold-grade or equivalent, free from splits, large knots.',
    'Scaffold planks must overlap 12 inches or extend 6 inches past support.',
    'serious',
    '1926.451'
),
(
    '1926.451(c)(1)',
    'Supported Scaffold Stability',
    'Scaffolding',
    'Supported Scaffolds',
    'Supported scaffolds with a height to base width ratio of more than 4:1 shall be restrained from tipping by guying, tying, bracing, or equivalent means. Guys, ties, braces shall be installed at locations where horizontal scaffold components support both inner and outer legs. Prevents scaffold from tipping over.',
    'Tall scaffolds (height >4x base width) must be tied, guyed, or braced.',
    'critical',
    '1926.451'
),
(
    '1926.451(c)(2)',
    'Supported Scaffold Poles and Legs',
    'Scaffolding',
    'Supported Scaffolds',
    'Supported scaffold poles, legs, posts, frames, and uprights shall bear on base plates and mud sills or other adequate firm foundation. Footings shall be level, sound, rigid, and capable of supporting loaded scaffold without settling or displacement. Prevents scaffold sinking into ground or soft surfaces.',
    'Scaffold legs must rest on base plates, mudsills, and firm footing.',
    'critical',
    '1926.451'
),
(
    '1926.451(d)(1)',
    'Suspension Scaffold Rigging',
    'Scaffolding',
    'Suspension Scaffolds',
    'Suspension scaffold support devices shall rest on surfaces capable of supporting at least 4 times the load imposed by scaffold when operating at rated load. Direct connections to roofs and floors must be evaluated by qualified person to determine if they support intended loads. Wire rope used for scaffold suspension must be capable of supporting 6 times intended load.',
    'Suspension scaffold supports must handle 4x load; wire rope 6x load.',
    'critical',
    '1926.451'
),
(
    '1926.451(d)(3)',
    'Suspension Scaffold Platforms',
    'Scaffolding',
    'Suspension Scaffolds',
    'Platforms shall not extend beyond hanger or support at each end by more than 6 inches unless designed by qualified person. Platforms suspended by ropes must have guardrails installed on all open sides and ends. Anti-two-blocking devices required on hoists.',
    'Suspension scaffold platforms cannot extend beyond supports by more than 6 inches.',
    'serious',
    '1926.451'
),
(
    '1926.451(e)(1)',
    'Scaffold Access Requirements',
    'Scaffolding',
    'Access',
    'When scaffold platforms are more than 2 feet above or below a point of access, portable ladders, hook-on ladders, attachable ladders, stair towers, stairway-type ladders, ramps, walkways, integral prefabricated scaffold access, or direct access from another scaffold or structure shall be used. Cross-braces shall not be used as means of access.',
    'Scaffolds over 2 feet high need proper access (ladders/stairs), not cross-braces.',
    'serious',
    '1926.451'
),
(
    '1926.451(f)(3)',
    'Scaffold Use Prohibitions',
    'Scaffolding',
    'Use',
    'Scaffolds shall not be moved horizontally while employees are on them, unless scaffold designed by qualified person for such movement. Scaffolds shall not be used during storms or high winds unless part of fall protection system. Debris and materials must not be allowed to accumulate on platforms. Shore scaffolds obtained from another manufacturer must not be intermixed.',
    'Do not move scaffolds with workers on them or use during storms.',
    'serious',
    '1926.451'
),
(
    '1926.451(f)(7)',
    'Scaffold Plumb and Level',
    'Scaffolding',
    'Use',
    'Scaffolds shall be erected, moved, dismantled, or altered only under supervision of competent person. Tubular welded frame scaffolds must be plumb and level. Footings must be level and capable of supporting scaffold loads. Front-end loaders and similar equipment shall not be used to support scaffold platforms unless specifically designed for such use.',
    'Scaffolds must be plumb, level, and erected under competent person supervision.',
    'serious',
    '1926.451'
),
(
    '1926.451(g)(1)',
    'Scaffold Fall Protection Requirements',
    'Scaffolding',
    'Fall Protection',
    'Each employee on a scaffold more than 10 feet above a lower level shall be protected from falling to that lower level by use of guardrails or personal fall arrest systems. Guardrail system or personal fall arrest required above 10 feet. Applies to all scaffold types including supported scaffolds, suspension scaffolds, aerial lifts.',
    'Workers on scaffolds over 10 feet high need guardrails or fall arrest.',
    'critical',
    '1926.451'
),
(
    '1926.451(g)(2)',
    'Scaffold Guardrail Height',
    'Scaffolding',
    'Fall Protection',
    'Guardrail systems installed to meet requirements of this section shall comply with 1926.502(b). Top edge height of toprails must be 42 inches plus or minus 3 inches above platform surface. Midrails must be midway between toprail and platform. Toeboards required when persons can pass beneath scaffold, or when tools, materials, or equipment could fall on persons below.',
    'Scaffold guardrails must be 39-45 inches high with midrails and toeboards.',
    'serious',
    '1926.451'
),
(
    '1926.451(g)(4)',
    'Scaffold Crossbracing as Fall Protection',
    'Scaffolding',
    'Fall Protection',
    'Crossbracing is acceptable in place of a midrail when crossing serves as midrail and meets required criteria. Crossbracing shall be installed between scaffold uprights. Ends of all rails shall not overhang terminal posts except when necessary for overlapping. Diagonal bracing in both directions required for tubular frame scaffolds.',
    'Crossbracing can serve as midrail if it meets height requirements.',
    'serious',
    '1926.451'
),
(
    '1926.451(h)(1)',
    'Falling Object Protection',
    'Scaffolding',
    'Falling Objects',
    'Overhead protection shall be provided for persons on a scaffold exposed to overhead hazards. Canopy structures, screens, guardrails, or debris nets shall be provided where employees may be struck by falling objects. Toeboards, screens, guardrail systems, canopies, or barricades required to protect from falling objects.',
    'Provide overhead protection where workers could be struck by falling objects.',
    'serious',
    '1926.451'
),
(
    '1926.451(h)(2)',
    'Scaffold Toeboards',
    'Scaffolding',
    'Falling Objects',
    'Toeboards shall be installed on scaffolds where employees below could be struck by falling objects. Minimum toeboard height 3.5 inches. Toeboards must be substantial enough to withstand upward force of 50 pounds. Includes toe boards on platform edges, around holes, and at open sides.',
    'Toeboards (minimum 3.5 inches) required where objects could fall on workers below.',
    'serious',
    '1926.451'
),
(
    '1926.452(a)',
    'Pole Scaffolds',
    'Scaffolding',
    'Scaffold Types',
    'Pole scaffolds shall be constructed and arranged so that loads are supported by bearers placed directly over or adjacent to posts. Maximum distance between bearer centers is 5 feet. Pole scaffolds more than 60 feet high shall be designed by qualified person. Guardrails, midrails, and toeboards required on all open sides and ends.',
    'Pole scaffolds over 60 feet need qualified person design.',
    'serious',
    '1926.452'
),
(
    '1926.452(b)',
    'Tube and Coupler Scaffolds',
    'Scaffolding',
    'Scaffold Types',
    'Tube and coupler scaffolds shall be constructed and arranged so that all tube and coupler connections are secured. Posts, runners, and bearers must be constructed of structural pipe. Couplers must be slip-type, swivel, or fixed. All couplers must be tightened with wrench to manufacturer specifications.',
    'Tube and coupler scaffold connections must be properly tightened.',
    'serious',
    '1926.452'
),
(
    '1926.452(c)',
    'Fabricated Frame Scaffolds',
    'Scaffolding',
    'Scaffold Types',
    'Fabricated frame scaffolds shall be erected with all brace connections secured. Frames and accessories shall not be intermixed or interchanged between manufacturers unless parts fit together without force and scaffold structural integrity is maintained. Uplift protection required on all supported scaffolds. Locking pins, clips, or other positive locking devices required.',
    'Frame scaffold components must be secured and not mixed between manufacturers.',
    'serious',
    '1926.452'
),
(
    '1926.452(w)',
    'Mobile Scaffolds',
    'Scaffolding',
    'Mobile Scaffolds',
    'Mobile scaffolds shall be designed by qualified person and constructed to sustain 4 times intended load. Caster stems shall be pinned or otherwise secured in scaffold legs or adjustment screws. Scaffold casters and wheels shall be locked to prevent movement during use. Employees shall not be allowed to ride on scaffolds unless certain conditions met (level, no obstructions, no holes, within floor load limits, manual force to move).',
    'Mobile scaffolds must have locked casters; workers only ride if conditions safe.',
    'serious',
    '1926.452'
),
(
    '1926.453(a)(1)',
    'Aerial Lifts General Requirements',
    'Scaffolding',
    'Aerial Lifts',
    'Aerial lifts acquired for use shall be designed and constructed to comply with applicable requirements of ANSI A92.2. Aerial devices include boom platforms, aerial ladders, articulating boom platforms, vertical towers, vehicle-mounted elevating and rotating platforms. Only trained persons shall operate aerial lifts.',
    'Aerial lifts must meet ANSI standards and only be operated by trained workers.',
    'serious',
    '1926.453'
),
(
    '1926.453(b)(1)',
    'Aerial Lift Specific Requirements',
    'Scaffolding',
    'Aerial Lifts',
    'Lift controls shall be tested each day prior to use to determine that such controls are in safe working condition. Only authorized persons shall operate an aerial lift. Belting off to adjacent pole, structure, or equipment while working from aerial lift shall not be permitted. Body belt shall be worn and lanyard attached to boom or basket when working from aerial lift.',
    'Aerial lifts must be tested daily; workers must use fall protection attached to boom.',
    'serious',
    '1926.453'
),
(
    '1926.454(a)',
    'Scaffold Training Requirements',
    'Scaffolding',
    'Training',
    'Employer shall have each employee who performs work while on a scaffold trained by person qualified to recognize hazards and train employees to minimize them. Training must cover nature of hazards, correct procedures for erecting, disassembling, moving, operating, inspecting scaffolds, proper use of fall protection systems, electrical hazards, capacity limits.',
    'All scaffold workers must receive training on hazards and safe use.',
    'serious',
    '1926.454'
),
(
    '1926.454(b)',
    'Scaffold Retraining',
    'Scaffolding',
    'Training',
    'Retraining required when changes in workplace render previous training obsolete, changes in scaffold type make training obsolete, inadequacies in affected employee knowledge or use indicate retraining needed, or employee has not retained requisite proficiency.',
    'Scaffold retraining required when workplace changes or worker knowledge inadequate.',
    'serious',
    '1926.454'
),

-- LADDERS AND STAIRWAYS (Subpart X)
(
    '1926.1051(a)',
    'Stairways and Ladders General',
    'Ladders',
    'General',
    'Stairway or ladder shall be provided at all personnel points of access where there is a break in elevation of 19 inches or more, and no ramp, runway, sloped embankment, or personnel hoist is provided. When there is only one point of access between levels, it shall be kept clear. Applies to all construction work areas.',
    'Provide stairs or ladders at elevation changes of 19+ inches.',
    'serious',
    '1926.1051'
),
(
    '1926.1052(a)(1)',
    'Stairway Construction Requirements',
    'Ladders',
    'Stairways',
    'Stairways that will not be a permanent part of structure shall have landings of not less than 30 inches in direction of travel and extend 22 inches in width at every 12 feet or less of vertical rise. Stairs must be installed at 30-50 degrees from horizontal. Stair treads and landings must be slip-resistant.',
    'Temporary stairs need landings every 12 feet, 30x22 inches minimum.',
    'serious',
    '1926.1052'
),
(
    '1926.1052(b)',
    'Stairrails and Handrails',
    'Ladders',
    'Stairways',
    'Stairways having four or more risers or rising more than 30 inches shall be equipped with at least one handrail and one stairrail system along each unprotected side or edge. Handrails and top rails must be capable of withstanding 200 pounds force applied in any direction. Height of stairrail systems must be 36 inches from surface of tread.',
    'Stairs with 4+ risers or over 30 inches need handrails and stairrails.',
    'serious',
    '1926.1052'
),
(
    '1926.1052(c)(1)',
    'Temporary Stairways',
    'Ladders',
    'Stairways',
    'Except during construction of permanent stairs, skeleton metal frame structures and steps shall be solidly filled in with solid material. Temporary treads shall be made of wood or other solid material, installed length of stair. Maximum riser height 9.5 inches. Minimum tread depth 9.5 inches. Variations in riser height or tread depth shall not exceed 1/4 inch.',
    'Temporary stairs must have solid treads/risers with uniform dimensions.',
    'serious',
    '1926.1052'
),
(
    '1926.1053(a)(1)',
    'Ladder General Requirements',
    'Ladders',
    'General',
    'Ladders shall be capable of supporting maximum intended loads without failure. Portable ladders shall be used at pitch no greater than 75 degrees from horizontal. Ladders shall be used only for purpose for which designed. Non-self-supporting ladders shall be used at angle where horizontal distance from top support to foot of ladder is 1/4 working length of ladder.',
    'Ladders must support intended loads and use 4:1 ratio (1 foot out per 4 feet up).',
    'serious',
    '1926.1053'
),
(
    '1926.1053(a)(2)',
    'Ladder Surfaces',
    'Ladders',
    'General',
    'Ladder rungs, cleats, and steps shall be parallel, level, and uniformly spaced when ladder is in position for use. Rungs shall be spaced not less than 10 inches apart and not more than 14 inches apart, measured between centerlines of rungs, cleats, and steps. Non-slip safety devices may be used.',
    'Ladder rungs must be parallel, level, spaced 10-14 inches apart.',
    'serious',
    '1926.1053'
),
(
    '1926.1053(a)(3)',
    'Ladder Structural Defects',
    'Ladders',
    'General',
    'Ladders shall be inspected by competent person for visible defects on periodic basis and after any occurrence that could affect their safe use. Portable ladders with structural defects shall be immediately marked in manner that readily identifies them as defective, or tagged with "Do Not Use" or similar language, and withdrawn from service until repaired.',
    'Inspect ladders regularly; remove defective ladders from service immediately.',
    'serious',
    '1926.1053'
),
(
    '1926.1053(b)(1)',
    'Ladder Use Requirements - Positioning',
    'Ladders',
    'Use',
    'When portable ladders are used for access to upper landing surface, side rails shall extend at least 3 feet above upper landing surface to which ladder is used to gain access. When such extension is not possible due to ladder length, ladder shall be secured at top and grab device shall be provided to assist in mounting and dismounting.',
    'Ladder side rails must extend 3 feet above landing for safe mounting/dismounting.',
    'serious',
    '1926.1053'
),
(
    '1926.1053(b)(4)',
    'Ladder Securing Requirements',
    'Ladders',
    'Use',
    'Ladders shall be used only on stable and level surfaces unless secured to prevent accidental displacement. Ladders placed in areas such as passageways, doorways, driveways, or where they can be displaced by workplace activities or traffic shall be secured or barricaded. Top and bottom must be secured or have someone holding ladder.',
    'Secure ladders or use on stable/level surfaces; barricade high-traffic areas.',
    'serious',
    '1926.1053'
),
(
    '1926.1053(b)(5)',
    'Ladder Movement Prohibition',
    'Ladders',
    'Use',
    'Ladders shall not be moved, shifted, or extended while occupied. Extension ladder sections must be secured together. Ladders must not be loaded beyond maximum intended load. Do not move ladder while worker on it; worker must descend first.',
    'Never move, shift, or extend ladders while workers are on them.',
    'serious',
    '1926.1053'
),
(
    '1926.1053(b)(6)',
    'Ladder Load Limits',
    'Ladders',
    'Use',
    'Ladders shall be used only for purpose for which designed. Non-self-supporting ladders shall be used at angle such that horizontal distance from top support to foot of ladder is approximately one-quarter of working length of ladder. Fixed ladders must support at least two loads of 250 pounds each. Portable ladders designed to specific duty ratings.',
    'Do not exceed ladder load rating or use ladders improperly.',
    'serious',
    '1926.1053'
),
(
    '1926.1053(b)(12)',
    'Ladder Face Requirements',
    'Ladders',
    'Use',
    'Ladders shall be used only on stable and level surfaces unless secured. Employee shall face ladder when ascending or descending. Each employee shall use at least one hand to grasp ladder when climbing. Maintain three-point contact (two hands and foot, or two feet and hand).',
    'Face ladder when climbing; maintain three-point contact.',
    'serious',
    '1926.1053'
),
(
    '1926.1053(b)(13)',
    'Top Step Prohibition',
    'Ladders',
    'Use',
    'Employee shall not stand on top two rungs or steps of stepladder. Top of stepladder is not designed to be stood on. Standing on top creates fall hazard due to imbalance and lack of support. Use taller ladder if additional height needed.',
    'Do not stand on top two rungs/steps of a stepladder.',
    'serious',
    '1926.1053'
),
(
    '1926.1053(b)(15)',
    'Ladder Single Rail Use',
    'Ladders',
    'Use',
    'Ladders shall have a minimum clear distance of 7 inches from centerline of rungs to nearest permanent object behind ladder. Step across distance of more than 12 inches requires landing platform. Single rail ladders shall not be used. Job-made ladders must be constructed for intended use.',
    'Single-rail ladders prohibited; maintain 7-inch clearance behind ladder.',
    'serious',
    '1926.1053'
),
(
    '1926.1053(b)(22)',
    'Ladder Electrical Hazards',
    'Ladders',
    'Use',
    'Portable metal ladders shall not be used for electrical work or where ladder or person using ladder could contact exposed energized electrical equipment. Use non-conductive fiberglass or wood ladders near electrical hazards. Applies to work on electrical panels, overhead lines, energized equipment.',
    'Use non-conductive ladders near electrical equipment; no metal ladders.',
    'critical',
    '1926.1053'
),
(
    '1926.1060(a)',
    'Ladder and Stairway Training',
    'Ladders',
    'Training',
    'Employer shall provide training for each employee using ladders and stairways. Program shall enable each employee to recognize hazards related to ladders and stairways and use proper procedures to minimize these hazards. Training must cover nature of fall hazards, correct procedures for erecting, maintaining, and disassembling fall protection, proper construction, use, placement, care of ladders and stairways.',
    'Train all workers on ladder/stairway hazards and safe use procedures.',
    'serious',
    '1926.1060'
),

-- EXCAVATIONS (Subpart P)
(
    '1926.651(a)',
    'Surface Encumbrances',
    'Excavations',
    'General',
    'Surface encumbrances located so as to create hazard to employees shall be removed or supported to prevent cave-ins. Includes trees, boulders, sidewalks, utilities, pavements positioned such that they could fall or roll into excavation. Must be removed, relocated, or adequately supported before excavating.',
    'Remove or support surface objects that could fall into excavation.',
    'serious',
    '1926.651'
),
(
    '1926.651(b)(1)',
    'Underground Installations',
    'Excavations',
    'General',
    'Location of utility installations such as sewer, telephone, fuel, electric, water lines or other underground installations that may be encountered during excavation work shall be determined prior to opening excavation. Utility companies shall be contacted and advised of proposed work prior to start. When excavation approaches estimated location of underground installations, determine exact location by safe and acceptable means.',
    'Call utility locators before digging; determine exact location of underground utilities.',
    'critical',
    '1926.651'
),
(
    '1926.651(c)(1)',
    'Excavation Access and Egress',
    'Excavations',
    'Access',
    'Means of egress from trench excavations shall be provided where employees may be exposed to vehicular traffic or cave-ins. Structural ramp used for access shall be designed by competent person. Ramps and runways constructed of two or more structural members must have cleats or other connecting members spaced at 4-foot intervals. Ladder, stairway, ramp, or other means of egress required in trench 4 feet deep or greater.',
    'Trenches 4+ feet deep need ladder/ramp within 25 feet of workers.',
    'serious',
    '1926.651'
),
(
    '1926.651(c)(2)',
    'Excavation Egress Spacing',
    'Excavations',
    'Access',
    'Means of egress shall be located so as to require no more than 25 feet of lateral travel for employees in excavation. In trenches 4 feet or deeper, provide ladder, stairway, or ramp every 25 feet of trench length. Worker must be able to reach exit within 25 feet.',
    'Excavation exits must be within 25 feet lateral travel for all workers.',
    'serious',
    '1926.651'
),
(
    '1926.651(d)',
    'Excavation Exposure to Vehicular Traffic',
    'Excavations',
    'Traffic Hazards',
    'When employees are exposed to public vehicular traffic, employees shall be provided with and wear warning vests or other suitable garments marked with or made of reflectorized or high-visibility material. Stop logs, barricades, or other surface crossing protection must be provided where vehicles will cross over excavations.',
    'Workers exposed to traffic need high-visibility vests; barricade excavations.',
    'serious',
    '1926.651'
),
(
    '1926.651(e)',
    'Excavation Exposure to Falling Loads',
    'Excavations',
    'Falling Hazards',
    'Employees shall be protected from excavated or other materials or equipment that could pose hazard by falling or rolling into excavations. Protection shall be provided by placing materials at least 2 feet from edge of excavations, or using retaining devices, or using combination of both if needed. Spoil piles, materials, equipment must be kept back from edge.',
    'Keep excavated materials and equipment at least 2 feet from excavation edge.',
    'serious',
    '1926.651'
),
(
    '1926.651(f)',
    'Excavation Warning Systems',
    'Excavations',
    'Warning Systems',
    'Warning system such as barricades, hand or mechanical signals, or stop logs shall be utilized when mobile equipment is operated adjacent to excavation or when such equipment is required to approach edge of excavation and operator does not have clear view of edge. Prevents vehicles from falling into excavation.',
    'Barricade excavations or use warning systems when vehicles operate nearby.',
    'serious',
    '1926.651'
),
(
    '1926.651(g)(1)',
    'Excavation Hazardous Atmospheres',
    'Excavations',
    'Atmospheric Hazards',
    'Testing and controls required when oxygen deficiency or hazardous atmosphere exists or could reasonably be expected to exist. Atmospheric testing required before employees enter excavations greater than 4 feet deep where oxygen deficiency or hazardous atmosphere exists or could be expected. Test for oxygen content, flammable gases, toxic gases.',
    'Test atmosphere in excavations 4+ feet deep for oxygen, toxic gases, flammables.',
    'critical',
    '1926.651'
),
(
    '1926.651(h)(1)',
    'Excavation Water Accumulation',
    'Excavations',
    'Water Hazards',
    'Employees shall not work in excavations in which there is accumulated water or in excavations in which water is accumulating unless adequate precautions taken. If water is controlled by special equipment such as water removal equipment, equipment and operation shall be monitored by competent person. Water removal equipment must be attended at all times.',
    'Control water accumulation in excavations; monitor pumping equipment continuously.',
    'serious',
    '1926.651'
),
(
    '1926.651(i)(1)',
    'Stability of Adjacent Structures',
    'Excavations',
    'Adjacent Structures',
    'Where stability of adjoining buildings, walls, or other structures is endangered by excavation operations, support systems such as shoring, bracing, or underpinning shall be provided to ensure stability of such structures for protection of employees. Buildings, sidewalks, pavements, utilities adjacent to excavation must be supported and protected.',
    'Shore or brace structures adjacent to excavations to prevent collapse.',
    'critical',
    '1926.651'
),
(
    '1926.651(j)(2)',
    'Protection from Loose Rock or Soil',
    'Excavations',
    'Cave-in Protection',
    'Adequate protection shall be provided to protect employees from loose rock or soil that could pose hazard by falling or rolling from face of excavation. Protection shall be provided by scaling to remove loose material or installation of protective barricade or other equivalent means. Applies to unstable soil conditions, rock faces.',
    'Scale or barricade excavations to protect from falling/rolling rock and soil.',
    'serious',
    '1926.651'
),
(
    '1926.651(k)(1)',
    'Daily Excavation Inspections',
    'Excavations',
    'Inspections',
    'Daily inspections of excavations, adjacent areas, and protective systems shall be made by competent person for evidence of situation that could result in cave-ins, indications of failure of protective systems, hazardous atmospheres, or other hazardous conditions. Inspections required prior to start of work and as needed throughout shift. After rainstorms or other hazard-increasing occurrences, excavation must be inspected.',
    'Competent person must inspect excavations daily and after rain/hazard events.',
    'serious',
    '1926.651'
),
(
    '1926.652(a)(1)',
    'Protection Systems Required',
    'Excavations',
    'Protection Systems',
    'Each employee in excavation shall be protected from cave-ins by adequate protective system except when excavations made entirely in stable rock or excavations less than 5 feet deep and examination by competent person provides no indication of potential cave-in. Protective systems include sloping, shoring, shielding. Required for excavations 5 feet or deeper.',
    'Excavations 5+ feet deep require cave-in protection (sloping, shoring, or shielding).',
    'critical',
    '1926.652'
),
(
    '1926.652(b)',
    'Excavation Sloping Requirements',
    'Excavations',
    'Protection Systems',
    'Maximum allowable slopes for excavations less than 20 feet based on soil type. Stable rock: vertical (90 degrees). Type A soil: 3/4:1 (53 degrees). Type B soil: 1:1 (45 degrees). Type C soil: 1.5:1 (34 degrees). Short-term maximum allowable slopes of 1/2:1 for Type A soil only for excavations 12 feet or less. Competent person must determine soil type.',
    'Slope excavation sides based on soil type (Type A: 53°, Type B: 45°, Type C: 34°).',
    'critical',
    '1926.652'
),
(
    '1926.652(c)',
    'Timber Shoring Requirements',
    'Excavations',
    'Protection Systems',
    'Timber shoring systems shall be designed and constructed in accordance with Appendix C or other approved methods. Members of shoring system must be in good serviceable condition. Damaged materials must not be used. All shoring installed from top down and removed from bottom up. Wales, struts, sheeting must be properly sized and spaced.',
    'Timber shoring must follow approved design; install top-down, remove bottom-up.',
    'critical',
    '1926.652'
),
(
    '1926.652(d)',
    'Shield Systems (Trench Boxes)',
    'Excavations',
    'Protection Systems',
    'Shield systems shall be designed and constructed to support loads as determined by qualified person. Shields shall be installed in manner to provide protection from cave-ins. Employees shall be protected from cave-ins when entering/exiting shield areas. Shields can be either premanufactured or job-built according to 1926.652(c)(3) or (c)(4). Shield must not have lateral movement when installed. Bottom of shield must not extend more than 2 feet below grade.',
    'Trench boxes (shields) must be properly installed with no lateral movement.',
    'critical',
    '1926.652'
),

-- PERSONAL PROTECTIVE EQUIPMENT (Subpart E)
(
    '1926.95(a)',
    'PPE Hazard Assessment',
    'Personal Protective Equipment',
    'General',
    'Employer shall assess workplace to determine if hazards are present which necessitate use of personal protective equipment. If such hazards are present, employer shall select and have each affected employee use types of PPE that will protect from identified hazards. Includes assessment for head, eye/face, hand, foot, hearing, respiratory protection.',
    'Employers must assess workplace for PPE needs and provide appropriate equipment.',
    'serious',
    '1926.95'
),
(
    '1926.95(c)',
    'PPE Defective Equipment',
    'Personal Protective Equipment',
    'General',
    'Defective and damaged personal protective equipment shall not be used. Equipment found to be defective during inspection must be removed from service and replaced. Includes cracked hard hats, damaged safety glasses, torn gloves, defective respirators.',
    'Remove defective PPE from service; replace damaged equipment.',
    'serious',
    '1926.95'
),
(
    '1926.100(a)',
    'Head Protection Requirements',
    'Personal Protective Equipment',
    'Head Protection',
    'Employees working in areas where there is possible danger of head injury from impact, falling or flying objects, or from electrical shock and burns shall be protected by protective helmets. Hard hats required when working below other workers, working with overhead hazards, working near exposed electrical conductors. Helmets must meet ANSI Z89.1 standards.',
    'Hard hats required where danger of head injury from impacts, falling objects, or electrical hazards.',
    'serious',
    '1926.100'
),
(
    '1926.100(b)',
    'Hard Hat Standards',
    'Personal Protective Equipment',
    'Head Protection',
    'Helmets for protection of employees exposed to high voltage electrical shock and burns shall meet specifications in ANSI Z89.2. Hard hats must be worn with bill forward unless manufacturer certifies use in reverse. Class G (general) protects up to 2,200 volts. Class E (electrical) protects up to 20,000 volts. Class C (conductive) provides no electrical protection.',
    'Use appropriate hard hat class for electrical hazards (Class E for high voltage).',
    'serious',
    '1926.100'
),
(
    '1926.101(b)',
    'Hearing Protection',
    'Personal Protective Equipment',
    'Hearing Protection',
    'Hearing protection shall be provided and used when sound levels exceed those in Table D-2 of 1926.52 or in any area where 8-hour time-weighted average exceeds 90 dBA. Ear plugs and/or ear muffs required. Plain cotton not acceptable. Must provide adequate attenuation for noise levels. Use properly fitted, maintained hearing protection.',
    'Provide hearing protection when noise exceeds 90 dBA (8-hour average).',
    'serious',
    '1926.101'
),
(
    '1926.102(a)(1)',
    'Eye and Face Protection General',
    'Personal Protective Equipment',
    'Eye and Face Protection',
    'Employees shall be provided with eye and face protection equipment when machines or operations present potential eye or face injury from physical, chemical, or radiation agents. Protection required when exposed to flying particles, molten metal, liquid chemicals, acids, caustic liquids, chemical gases, vapors, or injurious light radiation. Safety glasses, goggles, face shields, welding helmets as appropriate.',
    'Provide eye/face protection against flying particles, chemicals, radiation.',
    'serious',
    '1926.102'
),
(
    '1926.102(a)(2)',
    'Eye Protection Filter Lenses',
    'Personal Protective Equipment',
    'Eye and Face Protection',
    'Protection against radiant energy shall be provided when welding, cutting, and brazing operations present such hazards. Helmets or hand shields shall be used during all arc welding or arc cutting operations. Appropriate filter lenses required based on welding process and amperage. Helpers and attendants must also be protected by filter lenses.',
    'Welding requires helmets with appropriate filter lenses for arc intensity.',
    'serious',
    '1926.102'
),
(
    '1926.102(b)',
    'Eye Protection Criteria',
    'Personal Protective Equipment',
    'Eye and Face Protection',
    'Eye and face protection equipment shall meet requirements of ANSI Z87.1. Prescription lenses must meet same impact resistance as protective eyewear. Side shields required when hazard of flying objects exists. Equipment must be kept clean and in good repair. Defective equipment must be removed from service.',
    'Eye protection must meet ANSI Z87.1; use side shields for flying object hazards.',
    'serious',
    '1926.102'
),
(
    '1926.103',
    'Respiratory Protection',
    'Personal Protective Equipment',
    'Respiratory Protection',
    'Employer must provide respirators which are applicable and suitable for purpose intended when such equipment is necessary to protect health of employee. Requirements of 29 CFR 1910.134 shall be met. Includes medical evaluation, fit testing, training, maintenance. Applies to dust, fumes, mists, gases, vapors, oxygen-deficient atmospheres.',
    'Provide appropriate respirators per 1910.134 when air contaminants present.',
    'serious',
    '1926.103'
),
(
    '1926.104',
    'Safety Belts and Lanyards',
    'Personal Protective Equipment',
    'Body Protection',
    'Safety belts, lifelines, lanyards, and similar equipment shall be used only for employee safeguarding. Equipment must be protected from damage, inspected before each use, and defective components removed from service. Lifelines must be secured above point of operation and be free from sharp edges. Body belts not acceptable for fall arrest, only for positioning.',
    'Inspect lanyards/lifelines before each use; use only for worker protection.',
    'serious',
    '1926.104'
),
(
    '1926.105',
    'Safety Nets',
    'Personal Protective Equipment',
    'Safety Nets',
    'Safety nets shall be provided when workplaces are more than 25 feet above ground, water, or other surfaces where use of ladders, scaffolds, catch platforms, temporary floors, safety lines, or safety belts is impractical. Nets must extend 8 feet beyond edge of work surface where employees are exposed. Mesh size maximum 6 inches. Border rope minimum of 5,000 pounds tensile strength.',
    'Safety nets required over 25 feet when other fall protection impractical.',
    'serious',
    '1926.105'
),
(
    '1926.106',
    'Working Over or Near Water',
    'Personal Protective Equipment',
    'Water Hazards',
    'Employees working over or near water where danger of drowning exists shall be provided with Coast Guard approved life jacket or buoyant work vests. Ring buoys with at least 90 feet of line shall be provided and readily available for emergency rescue. At least one lifesaving skiff shall be immediately available at locations where employees are working over or adjacent to water.',
    'Provide life jackets and rescue equipment when working over/near water.',
    'serious',
    '1926.106'
),

-- ELECTRICAL SAFETY (Subpart K)
(
    '1926.403(a)',
    'Electrical Installation General',
    'Electrical Safety',
    'General Requirements',
    'All electrical conductors and equipment shall be approved. Conductors and equipment required to be approved shall be approved for specific purpose, function, use, environment, application. Listed, labeled, or certified equipment deemed approved. Includes wiring, devices, fixtures, apparatus, appliances.',
    'All electrical equipment must be approved for its specific use and environment.',
    'serious',
    '1926.403'
),
(
    '1926.403(b)',
    'Examination and Approval of Equipment',
    'Electrical Safety',
    'General Requirements',
    'Electrical equipment shall be free from recognized hazards that are likely to cause death or serious physical harm. Equipment must be examined by competent person or qualified person. Consider suitability for installation, mechanical strength, wire-bending and connection space, electrical insulation, heating effects, arcing effects, classification by type, rating.',
    'Competent person must examine electrical equipment for safety hazards.',
    'serious',
    '1926.403'
),
(
    '1926.404(a)(1)',
    'Wiring Design and Protection',
    'Electrical Safety',
    'Wiring Design',
    'Use and identification of grounded and grounding conductors required. Identification of grounded conductors by white or gray color. Equipment grounding conductors by continuous green color or green with yellow stripes. Grounding ensures protection from electrical shock.',
    'Properly identify grounded (white/gray) and grounding (green) conductors.',
    'serious',
    '1926.404'
),
(
    '1926.404(b)(1)',
    'Branch Circuits',
    'Electrical Safety',
    'Wiring Design',
    'Branch circuits recognized by this subpart shall be rated in accordance with maximum ampere rating or setting of overcurrent device. Multi-wire circuits must have means to disconnect simultaneously all ungrounded conductors. Prevents overloading circuits.',
    'Branch circuits must be properly rated with appropriate overcurrent protection.',
    'serious',
    '1926.404'
),
(
    '1926.404(f)(6)',
    'Grounding Requirements',
    'Electrical Safety',
    'Grounding',
    'Equipment connected by cord and plug shall be grounded. Exceptions for double-insulated tools. Non-current-carrying metal parts of equipment shall be grounded. Grounding protects against electrical shock from equipment faults. Three-prong plugs required for grounded equipment.',
    'Ground all cord-and-plug connected equipment unless double-insulated.',
    'critical',
    '1926.404'
),
(
    '1926.405(a)(2)',
    'Wiring Methods',
    'Electrical Safety',
    'Wiring Methods',
    'Conductors shall be spliced or joined with devices designed for use or by brazing, welding, or soldering with fusible metal or alloy. Soldered splices must first be mechanically and electrically secure without solder. All splices must be insulated. Wire nuts, crimp connectors, or other approved means required.',
    'Properly splice and insulate electrical connections with approved methods.',
    'serious',
    '1926.405'
),
(
    '1926.405(g)(1)',
    'Flexible Cords and Cables Use',
    'Electrical Safety',
    'Flexible Cords',
    'Flexible cords and cables shall be suitable for conditions of use and location. Used only for pendants, wiring of fixtures, connection of portable lamps, appliances, portable and mobile equipment, elevator cables, cranes and hoists, approved raceway systems. Not used as substitute for fixed wiring, run through holes in walls/ceilings/floors, attached to building surfaces, concealed.',
    'Extension cords only for temporary power; not substitute for permanent wiring.',
    'serious',
    '1926.405'
),
(
    '1926.405(g)(2)',
    'Flexible Cord Protection',
    'Electrical Safety',
    'Flexible Cords',
    'Flexible cords and cables shall be protected from damage. Sharp corners and projections avoided. Cords passing through doorways or other pinch points shall be protected. Pulling must be from plug, not cord. Cords shall not be fastened with staples, hung from nails, or suspended by wire.',
    'Protect extension cords from damage; do not staple, nail, or hang by wire.',
    'serious',
    '1926.405'
),
(
    '1926.405(j)(1)',
    'Temporary Wiring',
    'Electrical Safety',
    'Temporary Wiring',
    'Temporary electrical power and lighting installations 600 volts or less shall be allowed during periods of construction, remodeling, maintenance, repair, or demolition. Feeders must originate in approved distribution center. Conductors must be protected from damage. Disconnecting means required.',
    'Temporary wiring allowed during construction if properly protected and approved.',
    'serious',
    '1926.405'
),
(
    '1926.416(a)(1)',
    'Protection of Employees - No Live Parts',
    'Electrical Safety',
    'Safeguarding',
    'No employer shall permit employee to work in proximity to any part of electric power circuit that employee might contact unless employee is protected by de-energizing and grounding, or guarding by effective insulation. Live parts operating at 50 volts or more shall be guarded against accidental contact.',
    'De-energize, guard, or insulate live electrical parts before work.',
    'critical',
    '1926.416'
),
(
    '1926.416(e)(1)',
    'Handling of Portable Equipment',
    'Electrical Safety',
    'Safeguarding',
    'Portable equipment shall be handled in manner which will not cause damage. Flexible electric cords connected to equipment may not be used for raising or lowering equipment. Cannot pull on cord to disconnect from receptacle. Cords must not be fastened with staples or otherwise hung in manner that could damage outer jacket or insulation.',
    'Handle portable electrical equipment carefully; do not pull on cords.',
    'serious',
    '1926.416'
),
(
    '1926.417(a)',
    'Lockout and Tagging of Circuits',
    'Electrical Safety',
    'Lockout/Tagout',
    'Controls that are to be deactivated during course of work shall be tagged. Tags shall prohibit operation of disconnecting means. Tags must warn against hazards if machine or equipment is energized. Lock and tag required for circuits 600 volts or less. Energy control program required.',
    'Lock and tag out electrical circuits before work; prohibit re-energization.',
    'critical',
    '1926.417'
),
(
    '1926.431(a)',
    'Maintenance of Equipment',
    'Electrical Safety',
    'Maintenance',
    'Electrical equipment shall be free from recognized hazards likely to cause death or serious physical harm. Safety-related work practices shall be employed to prevent electric shock or other injuries resulting from direct or indirect contact. Equipment must be maintained in safe condition.',
    'Maintain electrical equipment free from hazards using safe work practices.',
    'serious',
    '1926.431'
),
(
    '1926.432(a)',
    'Environmental Deterioration',
    'Electrical Safety',
    'Environmental Conditions',
    'Unless identified for use in operating environment, no conductors or equipment shall be located in damp or wet locations, exposed to gases, fumes, vapors, liquids, corrosive conditions, or deteriorating agents that cause conductors or equipment to deteriorate. Protection against corrosion, physical damage required. Weatherproof equipment for outdoor use.',
    'Use electrical equipment rated for environmental conditions (wet, corrosive, etc.).',
    'serious',
    '1926.432'
),
(
    '1926.441(a)(1)',
    'GFCI Requirements for Receptacles',
    'Electrical Safety',
    'Ground-Fault Protection',
    'All 120-volt, single-phase, 15- and 20-ampere receptacle outlets on construction sites, which are not part of permanent wiring and used by employees, shall have approved ground-fault circuit interrupter protection for personnel or approved assured equipment grounding conductor program. GFCIs required for temporary power, extension cords, portable tools.',
    'All 120V construction receptacles need GFCI protection or assured grounding program.',
    'critical',
    '1926.441'
),

-- HOUSEKEEPING
(
    '1926.25(a)',
    'Housekeeping - General',
    'Housekeeping',
    'General',
    'Form and scrap lumber with protruding nails and all other debris shall be kept cleared from work areas, passageways, and stairs. Combustible scrap and debris shall be removed at regular intervals. Containers provided for collection and separation of waste, trash, oily and used rags, and other refuse. Keep work areas clean, organized, free of trip hazards.',
    'Keep work areas clear of debris, protruding nails, and combustible materials.',
    'other',
    '1926.25'
),
(
    '1926.25(b)',
    'Combustible Scrap',
    'Housekeeping',
    'Fire Prevention',
    'Combustible scrap and debris shall be removed at regular intervals during course of construction. Safe means shall be provided to facilitate such removal. Includes wood scraps, sawdust, oily rags, paper, packing materials. Prevents fire hazards and accumulation of flammable materials.',
    'Remove combustible debris regularly to prevent fire hazards.',
    'other',
    '1926.25'
),
(
    '1926.25(c)',
    'Containers for Waste',
    'Housekeeping',
    'Waste Disposal',
    'Containers shall be provided for collection and separation of waste, trash, oily and used rags, and other refuse. Containers used for garbage and other oily, flammable, or hazardous wastes shall be equipped with covers. Waste must be disposed of at frequent and regular intervals. Prevents pest infestation, fire hazards.',
    'Provide covered containers for waste; dispose of garbage regularly.',
    'other',
    '1926.25'
),
(
    '1926.20(b)(1)',
    'Accident Prevention Program',
    'Housekeeping',
    'Safety Programs',
    'It shall be responsibility of employer to initiate and maintain accident prevention program. Program shall provide for frequent and regular inspections of job sites, materials, and equipment by competent persons. Employer must initiate and maintain programs providing for frequent inspections.',
    'Employers must maintain accident prevention programs with regular inspections.',
    'serious',
    '1926.20'
),
(
    '1926.21(b)(2)',
    'Safety Instruction and Training',
    'Housekeeping',
    'Training',
    'Employer shall instruct each employee in recognition and avoidance of unsafe conditions and regulations applicable to work environment to control or eliminate hazards or exposure to illness or injury. Employees must be trained on hazards specific to their work. Training required before employee begins work.',
    'Train employees to recognize and avoid workplace hazards before starting work.',
    'serious',
    '1926.21'
),

-- HEAVY EQUIPMENT (Subpart O)
(
    '1926.600(a)',
    'Equipment General Requirements',
    'Heavy Equipment',
    'General',
    'All equipment left unattended at night, adjacent to highway in normal use, or adjacent to construction areas where work is in progress, shall have appropriate lights or reflectors, or barricades equipped with lights or reflectors. Equipment parked on inclines must have wheels chocked and parking brake set. Maintenance performed only when equipment stopped, controls neutralized.',
    'Barricade and light unattended equipment; chock wheels on inclines.',
    'serious',
    '1926.600'
),
(
    '1926.601(b)(1)',
    'Motor Vehicle Seat Belts',
    'Heavy Equipment',
    'Motor Vehicles',
    'Seat belts shall be provided on all equipment with rollover protective structures and operator-controlled equipment, except for equipment designed only for standup operation. Seatbelts must meet requirements of 49 CFR 571 (vehicle safety standards). Operators required to use seat belts.',
    'Equipment with ROPS must have seat belts; operators must use them.',
    'serious',
    '1926.601'
),
(
    '1926.601(b)(4)',
    'Equipment Access',
    'Heavy Equipment',
    'Motor Vehicles',
    'Tools and material shall be secured to prevent movement when transported in same compartment with employees. Vehicles used to transport employees shall have seats firmly secured. Employees shall not ride on loads that can shift or move. Passengers not permitted on heavy equipment unless seating and restraints provided.',
    'Secure tools and materials in vehicles; no riding on loads.',
    'serious',
    '1926.601'
),
(
    '1926.601(b)(6)',
    'Equipment Backing',
    'Heavy Equipment',
    'Motor Vehicles',
    'Heavy machinery, equipment, or parts thereof, being repaired shall be substantially blocked to prevent falling or shifting. When equipment operator has obstructed view to rear, ground observer or backing alarm required. Spotter must maintain visual contact with operator. Back-up alarms required on equipment.',
    'Use spotter or back-up alarm when equipment operator cannot see behind vehicle.',
    'serious',
    '1926.601'
),
(
    '1926.601(b)(14)',
    'Equipment Operator Visibility',
    'Heavy Equipment',
    'Motor Vehicles',
    'All vehicles with cabs shall be equipped with windshields and powered wipers. Cracked and broken glass shall be replaced. Windows may not be obstructed. Where operator vision is restricted, spotter required or backup alarm used. Operator must have clear visibility of work area.',
    'Equipment must have functioning windshields and wipers; replace broken glass.',
    'serious',
    '1926.601'
),
(
    '1926.602(a)(9)',
    'Rollover Protective Structures',
    'Heavy Equipment',
    'Earthmoving Equipment',
    'Rollover protective structures (ROPS) shall be provided on equipment listed in 1926.1000. ROPS required on dozers, scrapers, loaders, graders, crawler tractors, compactors manufactured after certain dates. Equipment must be fitted with ROPS meeting 1926.1001 or 1926.1002. Seat belts required with ROPS.',
    'Earthmoving equipment needs ROPS (rollover protection) and seat belts.',
    'critical',
    '1926.602'
),
(
    '1926.602(c)(1)',
    'Equipment Operating Procedures',
    'Heavy Equipment',
    'Operations',
    'Whenever vehicles are equipped with dump bodies, body shall be fully lowered or blocked when being repaired or when not in use. Operating levers controlling hoisting or dumping devices on haulage bodies shall be in position to prevent accidental starting. Employees not permitted under raised beds unless properly blocked.',
    'Lower or block raised dump bodies; never work under unblocked raised beds.',
    'critical',
    '1926.602'
),

-- MATERIAL HANDLING (Subpart H)
(
    '1926.250(a)(1)',
    'General Material Storage',
    'Material Handling',
    'Storage',
    'All materials stored in tiers shall be stacked, racked, blocked, interlocked, or otherwise secured to prevent sliding, falling or collapse. Storage areas must be kept free from accumulation of materials that constitute hazards from tripping, fire, explosion, or pest harborage. Aisles and passageways must be kept clear.',
    'Stack stored materials securely to prevent collapse; keep aisles clear.',
    'serious',
    '1926.250'
),
(
    '1926.250(a)(2)',
    'Storage Height Limits',
    'Material Handling',
    'Storage',
    'Bags, containers, bundles placed on top of each other shall be stepped back. Height limitations must be observed. Stack lumber no more than 16 feet high if handled manually; 20 feet if using forklift. Used lumber with nails must have nails removed before stacking. Prevent toppling.',
    'Stack materials with setbacks; observe height limits (lumber: 16-20 feet).',
    'serious',
    '1926.250'
),
(
    '1926.250(b)(1)',
    'Material Stacking and Storage',
    'Material Handling',
    'Storage',
    'Brick stacks shall not be more than 7 feet in height. Masonry blocks not stacked more than 6 feet high. Structural steel, poles, pipe, bar stock stored in racks or stacked and secured to prevent spreading. Lumber stacked on level and solidly supported sills. Stored materials must not create hazard.',
    'Stack bricks max 7 feet, blocks max 6 feet; secure steel and pipes in racks.',
    'serious',
    '1926.250'
),
(
    '1926.251(a)(1)',
    'Rigging Equipment General',
    'Material Handling',
    'Rigging',
    'Rigging equipment for material handling shall be inspected prior to use on each shift and as necessary during use to ensure that it is safe. Defective rigging equipment shall be removed from service. Includes slings, chains, ropes, hooks, shackles. Damaged rigging must not be used.',
    'Inspect rigging (slings, chains, hooks) before each shift; remove damaged equipment.',
    'serious',
    '1926.251'
),
(
    '1926.251(a)(5)',
    'Rigging Proof Testing',
    'Material Handling',
    'Rigging',
    'Job or shop-made rigging equipment shall be tested prior to use in accordance with applicable standards. Employer must maintain certification record including date of inspection, signature of person who performed inspection, serial number or identifier of rigging inspected.',
    'Proof-test custom rigging before use; maintain inspection records.',
    'serious',
    '1926.251'
),
(
    '1926.251(b)(2)',
    'Alloy Steel Chains',
    'Material Handling',
    'Rigging',
    'Welding and heat treating of alloy steel chain slings shall be performed only by sling manufacturer or equivalent. Chain slings shall be permanently marked with rated capacity. Chains with defects including cracked, bent, stretched links must be removed from service. Minimum sling length 30 inches.',
    'Do not weld or heat-treat chain slings; remove chains with damaged links.',
    'serious',
    '1926.251'
),
(
    '1926.251(c)(5)',
    'Wire Rope Inspection',
    'Material Handling',
    'Rigging',
    'Wire rope shall be inspected before each use and monthly when in regular use. Rope removed from service when containing six randomly distributed broken wires in one rope lay or three broken wires in one strand in one rope lay, wear exceeding 1/3 original diameter, kinking, crushing, bird caging, core protrusion, heat damage, end attachments cracked/deformed/worn.',
    'Inspect wire rope before use and monthly; remove if 6+ broken wires in one lay.',
    'serious',
    '1926.251'
),
(
    '1926.251(c)(9)',
    'Wire Rope End Attachments',
    'Material Handling',
    'Rigging',
    'Eyes in wire rope bridles, slings, or bull wires shall not be formed by wire rope clips or knots, except on haul back lines on scrapers. When U-bolt clips used for eye splices, Table H-20 specifies number and spacing of clips based on rope diameter. Thimbles required in eye terminations.',
    'Form wire rope eyes with proper clips (per Table H-20) and thimbles, not knots.',
    'serious',
    '1926.251'
),
(
    '1926.251(d)(3)',
    'Natural and Synthetic Fiber Rope',
    'Material Handling',
    'Rigging',
    'Fiber rope slings shall have minimum clear length of rope between eye splices equal to 10 times rope diameter. Knots shall not be used in lieu of splices. Fiber rope shall not be used if it contains defects including abnormal wear, powdered fiber, broken or cut fibers, variations in size, discoloration from heat or acids.',
    'Use spliced fiber rope slings; no knots. Remove ropes with wear or damage.',
    'serious',
    '1926.251'
),
(
    '1926.251(e)(1)',
    'Synthetic Web Slings',
    'Material Handling',
    'Rigging',
    'Sling identification stating rated capacities for types of hitches, type of material, and OSHA approval shall be on each synthetic web sling. Synthetic webbing shall be of uniform thickness and width. Selvage edges shall not be split from webbing. Repairs shall be made by sling manufacturer. Acid and caustic burns, melting, charring, snags, cuts require removal from service.',
    'Synthetic slings need capacity tags; remove slings with burns, cuts, or snags.',
    'serious',
    '1926.251'
),

-- +goose Down

DELETE FROM regulations WHERE standard_number LIKE '1926.%';
