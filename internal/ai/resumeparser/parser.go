package resumeparser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared/constant"
)

// ResumeParser handles resume parsing using OpenAI Vision
type ResumeParser struct {
	client *openai.Client
}

// NewResumeParser creates a new resume parser
func NewResumeParser(apiKey string) *ResumeParser {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &ResumeParser{
		client: &client,
	}
}

// ResumeData represents structured resume information
type ResumeData struct {
	PersonalInfo   PersonalInfo `json:"personal_info"`
	Summary        string       `json:"summary"`
	Skills         []string     `json:"skills"`
	Experience     []Experience `json:"experience"`
	Education      []Education  `json:"education"`
	Languages      []string     `json:"languages,omitempty"`
	Certifications []string     `json:"certifications,omitempty"`
}

type PersonalInfo struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
	LinkedIn string `json:"linkedin,omitempty"`
}

type Experience struct {
	Company          string   `json:"company"`
	Title            string   `json:"title"`
	StartDate        string   `json:"start_date"` // YYYY-MM format
	EndDate          string   `json:"end_date"`   // YYYY-MM or "Present"
	Responsibilities []string `json:"responsibilities"`
}

type Education struct {
	Institution    string `json:"institution"`
	Degree         string `json:"degree"`
	Field          string `json:"field"`
	GraduationDate string `json:"graduation_date"` // YYYY-MM format
	GPA            string `json:"gpa,omitempty"`
}

// ParseResumeFromImage parses a resume from image data (PDF page, JPG, PNG)
func (p *ResumeParser) ParseResumeFromImage(ctx context.Context, imageData []byte) (*ResumeData, error) {
	// Convert image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:image/jpeg;base64,%s", base64Image)

	// Structured extraction prompt
	systemPrompt := `You are a professional resume parser. Extract ALL information from the resume image and return ONLY valid JSON.`

	userPrompt := `Extract all information from this resume image in the following JSON structure:

{
  "personal_info": {
    "name": string,
    "email": string,
    "phone": string,
    "location": string,
    "linkedin": string (optional)
  },
  "summary": string (professional summary, max 250 words),
  "skills": string[] (technical and soft skills),
  "experience": [{
    "company": string,
    "title": string,
    "start_date": string (YYYY-MM format),
    "end_date": string (YYYY-MM or "Present"),
    "responsibilities": string[] (key achievements and duties)
  }],
  "education": [{
    "institution": string,
    "degree": string,
    "field": string,
    "graduation_date": string (YYYY-MM format),
    "gpa": string (optional)
  }],
  "languages": string[] (optional),
  "certifications": string[] (optional)
}

IMPORTANT:
- Extract ALL visible text accurately
- If a field is not available, omit it or use empty string
- Maintain chronological order (newest first)
- Return ONLY the JSON, no explanatory text
- Be thorough and precise`

	// Build messages with vision content
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
		{
			OfUser: &openai.ChatCompletionUserMessageParam{
				Content: openai.ChatCompletionUserMessageParamContentUnion{
					OfArrayOfContentParts: []openai.ChatCompletionContentPartUnionParam{
						{
							OfText: &openai.ChatCompletionContentPartTextParam{
								Type: constant.Text("text"),
								Text: userPrompt,
							},
						},
						{
							OfImageURL: &openai.ChatCompletionContentPartImageParam{
								Type: constant.ImageURL("image_url"),
								ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
									URL:    dataURL,
									Detail: "high", // High detail for better OCR
								},
							},
						},
					},
				},
			},
		},
	}

	// API call with JSON response format
	completion, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    "gpt-4o", // GPT-4o has best vision capabilities
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{
				Type: constant.JSONObject("json_object"),
			},
		},
		Temperature: openai.Float(0.1), // Low temperature for consistency
		MaxTokens:   openai.Int(4000),
	})

	if err != nil {
		return nil, fmt.Errorf("openai vision api error: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, errors.New("no response from openai")
	}

	// Parse JSON response
	content := completion.Choices[0].Message.Content
	var resumeData ResumeData
	if err := json.Unmarshal([]byte(content), &resumeData); err != nil {
		return nil, fmt.Errorf("failed to parse resume JSON: %w", err)
	}

	return &resumeData, nil
}

// ParseResumeFromMultiplePages parses a multi-page resume
func (p *ResumeParser) ParseResumeFromMultiplePages(ctx context.Context, pages [][]byte) (*ResumeData, error) {
	if len(pages) == 0 {
		return nil, errors.New("no pages provided")
	}

	// For single page, use standard parsing
	if len(pages) == 1 {
		return p.ParseResumeFromImage(ctx, pages[0])
	}

	// For multiple pages, send all images together
	systemPrompt := `You are a professional resume parser. This is a multi-page resume. Extract ALL information from ALL pages and return ONLY valid JSON.`

	userPrompt := `Extract all information from this multi-page resume in the following JSON structure:

{
  "personal_info": {
    "name": string,
    "email": string,
    "phone": string,
    "location": string,
    "linkedin": string (optional)
  },
  "summary": string,
  "skills": string[],
  "experience": [{
    "company": string,
    "title": string,
    "start_date": string (YYYY-MM),
    "end_date": string (YYYY-MM or "Present"),
    "responsibilities": string[]
  }],
  "education": [{
    "institution": string,
    "degree": string,
    "field": string,
    "graduation_date": string (YYYY-MM),
    "gpa": string (optional)
  }],
  "languages": string[],
  "certifications": string[]
}

Combine information from all pages. Return ONLY JSON.`

	// Build content parts with all pages
	contentParts := []openai.ChatCompletionContentPartUnionParam{
		{
			OfText: &openai.ChatCompletionContentPartTextParam{
				Type: constant.Text("text"),
				Text: userPrompt,
			},
		},
	}

	// Add all pages as images
	for i, pageData := range pages {
		base64Image := base64.StdEncoding.EncodeToString(pageData)
		dataURL := fmt.Sprintf("data:image/jpeg;base64,%s", base64Image)

		contentParts = append(contentParts, openai.ChatCompletionContentPartUnionParam{
			OfImageURL: &openai.ChatCompletionContentPartImageParam{
				Type: constant.ImageURL("image_url"),
				ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
					URL:    dataURL,
					Detail: "high",
				},
			},
		})

		// Add page separator text
		if i < len(pages)-1 {
			contentParts = append(contentParts, openai.ChatCompletionContentPartUnionParam{
				OfText: &openai.ChatCompletionContentPartTextParam{
					Type: constant.Text("text"),
					Text: fmt.Sprintf("--- Page %d ends, Page %d begins ---", i+1, i+2),
				},
			})
		}
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
		{
			OfUser: &openai.ChatCompletionUserMessageParam{
				Content: openai.ChatCompletionUserMessageParamContentUnion{
					OfArrayOfContentParts: contentParts,
				},
			},
		},
	}

	completion, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    "gpt-4o",
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{
				Type: constant.JSONObject("json_object"),
			},
		},
		Temperature: openai.Float(0.1),
		MaxTokens:   openai.Int(6000),
	})

	if err != nil {
		return nil, fmt.Errorf("openai vision api error: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, errors.New("no response from openai")
	}

	content := completion.Choices[0].Message.Content
	var resumeData ResumeData
	if err := json.Unmarshal([]byte(content), &resumeData); err != nil {
		return nil, fmt.Errorf("failed to parse resume JSON: %w", err)
	}

	return &resumeData, nil
}

// FormatResumeForEmbedding creates a text representation for embedding
func (rd *ResumeData) FormatResumeForEmbedding() string {
	var text string

	// Personal info
	text += fmt.Sprintf("Name: %s\n", rd.PersonalInfo.Name)
	if rd.PersonalInfo.Email != "" {
		text += fmt.Sprintf("Email: %s\n", rd.PersonalInfo.Email)
	}
	if rd.PersonalInfo.Location != "" {
		text += fmt.Sprintf("Location: %s\n", rd.PersonalInfo.Location)
	}

	// Summary
	if rd.Summary != "" {
		text += fmt.Sprintf("\nSummary: %s\n", rd.Summary)
	}

	// Skills
	if len(rd.Skills) > 0 {
		text += fmt.Sprintf("\nSkills: %s\n", joinStrings(rd.Skills, ", "))
	}

	// Experience
	if len(rd.Experience) > 0 {
		text += "\nExperience:\n"
		for _, exp := range rd.Experience {
			text += fmt.Sprintf("- %s at %s (%s to %s)\n", exp.Title, exp.Company, exp.StartDate, exp.EndDate)
			for _, resp := range exp.Responsibilities {
				text += fmt.Sprintf("  * %s\n", resp)
			}
		}
	}

	// Education
	if len(rd.Education) > 0 {
		text += "\nEducation:\n"
		for _, edu := range rd.Education {
			text += fmt.Sprintf("- %s in %s from %s (%s)\n", edu.Degree, edu.Field, edu.Institution, edu.GraduationDate)
		}
	}

	// Certifications
	if len(rd.Certifications) > 0 {
		text += fmt.Sprintf("\nCertifications: %s\n", joinStrings(rd.Certifications, ", "))
	}

	// Languages
	if len(rd.Languages) > 0 {
		text += fmt.Sprintf("\nLanguages: %s\n", joinStrings(rd.Languages, ", "))
	}

	return text
}

func joinStrings(arr []string, sep string) string {
	result := ""
	for i, s := range arr {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
