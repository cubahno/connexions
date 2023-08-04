package xs

import "reflect"

func ReplaceValues(target, replacements map[string]interface{}) {
    for key, val := range target {
        if repVal, ok := replacements[key]; ok {
            // Check if the replacement is a nested map
            if repMap, ok := repVal.(map[string]interface{}); ok {
                // If both target and replacement are nested maps, recursively call the function
                if targetMap, ok := val.(map[string]interface{}); ok {
                    ReplaceValues(targetMap, repMap)
                }
            } else if reflect.TypeOf(repVal) == reflect.TypeOf(val) || isNumeric(repVal) && isNumeric(val) {
                // Replace the value in target with the replacement value
                target[key] = repVal
            }
        } else if targetMap, ok := val.(map[string]interface{}); ok {
            // If the key is not present in replacements, check if the value is a nested map and call the function
            ReplaceValues(targetMap, replacements)
        }
    }
}

func isNumeric(value interface{}) bool {
    switch value.(type) {
    case int, int32, int64, float32, float64:
        return true
    default:
        return false
    }
}
