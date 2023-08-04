package xs

import (
    "encoding/json"
    "reflect"
    "testing"
)

func TestReplaceValues(t *testing.T) {
    tests := []struct {
        targetJSON       string
        replacementsJSON string
        expectedJSON     string
    }{
        // Test case 1
        {
            `{
				"key-1": {
					"key-2": {
						"key-3-1": 31,
						"key-3-2": 32
					}
				}
			}`,
            `{
				"key-3-2": 3232
			}`,
            `{
				"key-1": {
					"key-2": {
						"key-3-1": 31,
						"key-3-2": 3232
					}
				}
			}`,
        },

        // Test case 2
        {
            `{
				"key-1": {
					"key-2": {
						"key-3-1": 31,
						"key-3-2": 32,
						"key-3-3": 33
					}
				}
			}`,
            `{
				"key-2": {
					"key-3-1": 3131
				}
			}`,
            `{
				"key-1": {
					"key-2": {
						"key-3-1": 3131,
						"key-3-2": 32,
						"key-3-3": 33
					}
				}
			}`,
        },
        // Case 3. No replacements
        {
            `{
				"key-1": {
					"key-2": {
						"key-3-1": 31,
						"key-3-2": 32,
						"key-3-3": 33
					}
				}
			}`,
            `{}`,
            `{
				"key-1": {
					"key-2": {
						"key-3-1": 31,
						"key-3-2": 32,
						"key-3-3": 33
					}
				}
			}`,
        },
        // Case 4. Irrelevant replacements
        {
            `{
				"key-1": {
					"key-2": {
						"key-3-1": 31,
						"key-3-2": 32,
						"key-3-3": 33
					}
				}
			}`,
            `{
				"foo": {
					"bar": 3131
				}
			}`,
            `{
				"key-1": {
					"key-2": {
						"key-3-1": 31,
						"key-3-2": 32,
						"key-3-3": 33
					}
				}
			}`,
        },
        // Case 5. Wrong types
        {
            `{
				"key-1": {
					"key-2": {
						"key-3-1": 31,
						"key-3-2": "32",
						"key-3-3": 33
					}
				}
			}`,
            `{
				"key-2": {
					"key-3-2": 32
				}
			}`,
            `{
				"key-1": {
					"key-2": {
						"key-3-1": 31,
						"key-3-2": "32",
						"key-3-3": 33
					}
				}
			}`,
        },
    }

    for i, test := range tests {
        var target map[string]interface{}
        var replacements map[string]interface{}
        var expected map[string]interface{}

        if err := json.Unmarshal([]byte(test.targetJSON), &target); err != nil {
            t.Errorf("Test case %d: Error parsing target JSON: %v", i+1, err)
            continue
        }

        if err := json.Unmarshal([]byte(test.replacementsJSON), &replacements); err != nil {
            t.Errorf("Test case %d: Error parsing replacements JSON: %v", i+1, err)
            continue
        }

        if err := json.Unmarshal([]byte(test.expectedJSON), &expected); err != nil {
            t.Errorf("Test case %d: Error parsing expected JSON: %v", i+1, err)
            continue
        }

        ReplaceValues(target, replacements)

        if !reflect.DeepEqual(target, expected) {
            t.Errorf("Test case %d: Expected %v, but got %v", i+1, expected, target)
        }
    }
}
