package fal

import (
	"fmt"
	"time"
)

// ModelInfo represents information about a FAL AI model
type ModelInfo struct {
	Name        string             `json:"name"`
	DisplayName string             `json:"display_name"`
	Description string             `json:"description"`
	CostPerImage float64           `json:"cost_per_image"`
	Parameters  map[string]Parameter `json:"parameters"`
}

// Parameter represents a model parameter definition
type Parameter struct {
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
	Min         *float64    `json:"min,omitempty"`
	Max         *float64    `json:"max,omitempty"`
	Options     []string    `json:"options,omitempty"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
}

// GenerationRequest represents a request to generate images
type GenerationRequest struct {
	Model      string                 `json:"model"`
	Prompt     string                 `json:"prompt"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// GenerationResponse represents the response from FAL AI
type GenerationResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Images    []struct {
		URL         string `json:"url"`
		ThumbnailURL string `json:"thumbnail_url,omitempty"`
		Width       int    `json:"width,omitempty"`
		Height      int    `json:"height,omitempty"`
	} `json:"images"`
	Cost      float64                `json:"cost,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Error     *FALError              `json:"error,omitempty"`
}

// QueueResponse represents the initial queue response
type QueueResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
}

// StatusResponse represents a status check response
type StatusResponse struct {
	RequestID string                 `json:"request_id"`
	Status    string                 `json:"status"`
	Progress  float64                `json:"progress,omitempty"`
	ETA       *time.Duration         `json:"eta,omitempty"`
	Result    *GenerationResponse    `json:"result,omitempty"`
	Error     *FALError              `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// FALError represents an error from FAL AI
type FALError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *FALError) Error() string {
	return e.Message
}

// Status constants
const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"
)

// Supported models with their configurations
var SupportedModels = map[string]ModelInfo{
	"flux/schnell": {
		Name:         "flux/schnell",
		DisplayName:  "Flux Schnell",
		Description:  "Fast, high-quality image generation with Flux model",
		CostPerImage: 0.003,
		Parameters: map[string]Parameter{
			"image_size": {
				Type:        "object",
				Default:     "square_hd",
				Options:     []string{"square_hd", "square", "portrait_4_3", "portrait_16_9", "landscape_4_3", "landscape_16_9"},
				Description: "Image size as preset or custom dimensions object {width: int, height: int}",
				Required:    false,
			},
			"num_images": {
				Type:        "integer",
				Default:     1,
				Min:         floatPtr(1),
				Max:         floatPtr(4),
				Description: "Number of images to generate",
				Required:    false,
			},
			"guidance_scale": {
				Type:        "float",
				Default:     7.5,
				Min:         floatPtr(1.0),
				Max:         floatPtr(20.0),
				Description: "How closely to follow the prompt",
				Required:    false,
			},
			"num_inference_steps": {
				Type:        "integer",
				Default:     4,
				Min:         floatPtr(1),
				Max:         floatPtr(50),
				Description: "Number of denoising steps",
				Required:    false,
			},
			"seed": {
				Type:        "integer",
				Default:     nil,
				Description: "Random seed for reproducible results",
				Required:    false,
			},
		},
	},
	"hidream/hidream-i1-dev": {
		Name:         "hidream/hidream-i1-dev",
		DisplayName:  "HiDream I1 Dev",
		Description:  "High-quality image generation with HiDream model (development version)",
		CostPerImage: 0.004,
		Parameters: map[string]Parameter{
			"image_size": {
				Type:        "object",
				Default:     "square_hd",
				Options:     []string{"square_hd", "square", "portrait_4_3", "portrait_16_9", "landscape_4_3", "landscape_16_9"},
				Description: "Image size as preset or custom dimensions object {width: int, height: int}",
				Required:    false,
			},
			"num_images": {
				Type:        "integer",
				Default:     1,
				Min:         floatPtr(1),
				Max:         floatPtr(4),
				Description: "Number of images to generate",
				Required:    false,
			},
			"guidance_scale": {
				Type:        "float",
				Default:     7.5,
				Min:         floatPtr(1.0),
				Max:         floatPtr(20.0),
				Description: "How closely to follow the prompt",
				Required:    false,
			},
			"num_inference_steps": {
				Type:        "integer",
				Default:     20,
				Min:         floatPtr(10),
				Max:         floatPtr(100),
				Description: "Number of denoising steps",
				Required:    false,
			},
			"seed": {
				Type:        "integer",
				Default:     nil,
				Description: "Random seed for reproducible results",
				Required:    false,
			},
		},
	},
	"hidream/hidream-i1-fast": {
		Name:         "hidream/hidream-i1-fast",
		DisplayName:  "HiDream I1 Fast",
		Description:  "Fast image generation with HiDream model",
		CostPerImage: 0.003,
		Parameters: map[string]Parameter{
			"image_size": {
				Type:        "object",
				Default:     "square_hd",
				Options:     []string{"square_hd", "square", "portrait_4_3", "portrait_16_9", "landscape_4_3", "landscape_16_9"},
				Description: "Image size as preset or custom dimensions object {width: int, height: int}",
				Required:    false,
			},
			"num_images": {
				Type:        "integer",
				Default:     1,
				Min:         floatPtr(1),
				Max:         floatPtr(4),
				Description: "Number of images to generate",
				Required:    false,
			},
			"guidance_scale": {
				Type:        "float",
				Default:     7.5,
				Min:         floatPtr(1.0),
				Max:         floatPtr(15.0),
				Description: "How closely to follow the prompt",
				Required:    false,
			},
			"num_inference_steps": {
				Type:        "integer",
				Default:     8,
				Min:         floatPtr(4),
				Max:         floatPtr(20),
				Description: "Number of denoising steps",
				Required:    false,
			},
			"seed": {
				Type:        "integer",
				Default:     nil,
				Description: "Random seed for reproducible results",
				Required:    false,
			},
		},
	},
}

// GetModel returns model information by name
func GetModel(name string) (ModelInfo, bool) {
	model, exists := SupportedModels[name]
	return model, exists
}

// GetAllModels returns all supported models
func GetAllModels() map[string]ModelInfo {
	return SupportedModels
}

// ValidateParameters validates generation parameters against model requirements
func (m *ModelInfo) ValidateParameters(params map[string]interface{}) error {
	for key, value := range params {
		param, exists := m.Parameters[key]
		if !exists {
			continue // Allow unknown parameters (they'll be ignored by FAL)
		}

		// Type validation
		switch param.Type {
		case "integer":
			if _, ok := value.(int); !ok {
				if f, ok := value.(float64); ok && f == float64(int(f)) {
					// Allow float64 that represents an integer
					params[key] = int(f)
				} else {
					return &FALError{
						Code:    "invalid_parameter_type",
						Message: key + " must be an integer",
					}
				}
			}
		case "float":
			if _, ok := value.(float64); !ok {
				if i, ok := value.(int); ok {
					// Allow int that can be converted to float
					params[key] = float64(i)
				} else {
					return &FALError{
						Code:    "invalid_parameter_type",
						Message: key + " must be a number",
					}
				}
			}
		case "string":
			if _, ok := value.(string); !ok {
				return &FALError{
					Code:    "invalid_parameter_type",
					Message: key + " must be a string",
				}
			}
		case "object":
			// Special handling for image_size parameter
			if key == "image_size" {
				// Can be either a string (enum) or an object with width/height
				if strValue, ok := value.(string); ok {
					// Validate against enum options
					if len(param.Options) > 0 {
						valid := false
						for _, option := range param.Options {
							if strValue == option {
								valid = true
								break
							}
						}
						if !valid {
							return &FALError{
								Code:    "invalid_parameter_value",
								Message: key + " must be one of: " + joinStrings(param.Options, ", ") + " or an object with width and height",
							}
						}
					}
				} else if objValue, ok := value.(map[string]interface{}); ok {
					// Validate object has width and height
					width, hasWidth := objValue["width"]
					height, hasHeight := objValue["height"]
					
					if !hasWidth || !hasHeight {
						return &FALError{
							Code:    "invalid_parameter_value",
							Message: key + " object must have both 'width' and 'height' properties",
						}
					}
					
					// Validate width and height are integers
					if _, ok := width.(int); !ok {
						if f, ok := width.(float64); ok && f == float64(int(f)) {
							objValue["width"] = int(f)
						} else {
							return &FALError{
								Code:    "invalid_parameter_type",
								Message: key + ".width must be an integer",
							}
						}
					}
					
					if _, ok := height.(int); !ok {
						if f, ok := height.(float64); ok && f == float64(int(f)) {
							objValue["height"] = int(f)
						} else {
							return &FALError{
								Code:    "invalid_parameter_type",
								Message: key + ".height must be an integer",
							}
						}
					}
				} else {
					return &FALError{
						Code:    "invalid_parameter_type",
						Message: key + " must be either a string (preset) or an object with width and height",
					}
				}
			} else {
				// Generic object validation
				if _, ok := value.(map[string]interface{}); !ok {
					return &FALError{
						Code:    "invalid_parameter_type",
						Message: key + " must be an object",
					}
				}
			}
		}

		// Range validation
		if param.Min != nil {
			var numValue float64
			switch v := value.(type) {
			case int:
				numValue = float64(v)
			case float64:
				numValue = v
			}
			if numValue < *param.Min {
				return &FALError{
					Code:    "parameter_out_of_range",
					Message: key + " must be at least " + floatToString(*param.Min),
				}
			}
		}

		if param.Max != nil {
			var numValue float64
			switch v := value.(type) {
			case int:
				numValue = float64(v)
			case float64:
				numValue = v
			}
			if numValue > *param.Max {
				return &FALError{
					Code:    "parameter_out_of_range",
					Message: key + " must be at most " + floatToString(*param.Max),
				}
			}
		}

		// Options validation (skip for image_size as it's handled specially above)
		if len(param.Options) > 0 && key != "image_size" {
			strValue, ok := value.(string)
			if !ok {
				return &FALError{
					Code:    "invalid_parameter_type",
					Message: key + " must be a string",
				}
			}
			valid := false
			for _, option := range param.Options {
				if strValue == option {
					valid = true
					break
				}
			}
			if !valid {
				return &FALError{
					Code:    "invalid_parameter_value",
					Message: key + " must be one of: " + joinStrings(param.Options, ", "),
				}
			}
		}
	}

	return nil
}

// Helper functions
func floatPtr(f float64) *float64 {
	return &f
}

func floatToString(f float64) string {
	if f == float64(int(f)) {
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%.2f", f)
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}