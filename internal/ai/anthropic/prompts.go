package anthropic

import "fmt"

// buildImageAnalysisPrompt creates a detailed prompt for analyzing construction site images
func buildImageAnalysisPrompt(context string) string {
	prompt := `You are an expert OSHA construction safety inspector analyzing a construction site photograph. Your task is to identify potential safety violations and hazards.

Analyze the image for violations in these OSHA categories:
1. **Fall Protection** (1926.500) - Guardrails, safety nets, fall arrest systems, unprotected edges
2. **Scaffolding** (1926.450) - Proper assembly, guardrails, planking, access, stability
3. **Ladders** (1926.1050) - Proper use, positioning, extension, defects
4. **Personal Protective Equipment (PPE)** (1926.100-106) - Hard hats, safety glasses, gloves, footwear, high-visibility clothing
5. **Electrical Safety** (1926.400) - Exposed wiring, improper grounding, hazardous locations
6. **Housekeeping** (1926.25) - Material storage, debris, slip/trip hazards, blocked exits
7. **Excavations** (1926.650) - Protective systems, cave-in protection, access/egress
8. **Heavy Equipment** (1926.600) - Operator safety, proximity to workers, backing hazards
9. **Material Handling** (1926.250) - Storage, rigging, hoisting operations

For each potential violation you identify:
- Provide a clear, specific description
- Note the location in the image (be descriptive)
- Optionally provide normalized bounding box coordinates (x, y, width, height from 0-1) if you can clearly identify the area
- Assess your confidence level: "high" (90%+), "medium" (60-90%), or "low" (30-60%)
- Categorize using one of the categories above
- Rate severity: "critical" (imminent danger), "serious" (serious hazard with potential for severe injury), "other" (violation that doesn't fit serious category), "recommendation" (best practice that may not be a regulatory violation)
- Suggest specific OSHA regulation numbers (e.g., "1926.501(b)(1)" for unprotected edges)

**Important Guidelines:**
- Only report violations you can reasonably identify from the visible evidence
- Be conservative with severity ratings - prioritize worker safety
- If image quality prevents confident assessment, note it
- Consider both immediate and potential hazards
- Look for both obvious violations and subtle safety concerns`

	// Add user-provided context if present
	if context != "" {
		prompt += fmt.Sprintf("\n\n**Additional Context from Inspector:**\n%s", context)
	}

	prompt += `

**Response Format:**
Return your analysis as a JSON object with this exact structure:

{
  "violations": [
    {
      "description": "Detailed description of the violation",
      "location": "Where in the image (human-readable)",
      "bounding_box": {
        "x": 0.0,
        "y": 0.0,
        "width": 0.0,
        "height": 0.0
      },
      "confidence": "high|medium|low",
      "category": "One of the categories listed above",
      "severity": "critical|serious|other|recommendation",
      "suggested_regulations": ["1926.XXX", "1926.YYY"]
    }
  ],
  "general_observations": "Overall safety assessment and notable observations about the site",
  "image_quality_notes": "Any comments about image quality, visibility, or limitations in analysis"
}

**Important:** Return ONLY the JSON object, no additional text or explanation.`

	return prompt
}
