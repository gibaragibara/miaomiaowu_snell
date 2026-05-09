package scriptengine

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

const defaultTimeout = 5 * time.Second

// RunPostFetch executes a "post_fetch" script against a full config map.
// The script must define: function main(config) { ... return config; }
func RunPostFetch(ctx context.Context, script string, config map[string]interface{}) (map[string]interface{}, error) {
	vm := goja.New()

	if err := vm.Set("__input__", config); err != nil {
		return nil, fmt.Errorf("set input: %w", err)
	}

	result, err := runWithTimeout(ctx, vm, script+";\nmain(__input__);")
	if err != nil {
		return nil, err
	}

	exported := result.Export()
	if exported == nil {
		return config, nil
	}

	resultMap, ok := exported.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("script must return an object, got %T", exported)
	}

	return resultMap, nil
}

// RunPreSaveNodes executes a "pre_save_nodes" script against a proxies array.
// The script must define: function main(proxies) { ... return proxies; }
func RunPreSaveNodes(ctx context.Context, script string, proxies []map[string]interface{}) ([]map[string]interface{}, error) {
	vm := goja.New()

	input := make([]interface{}, len(proxies))
	for i, p := range proxies {
		input[i] = p
	}

	if err := vm.Set("__input__", input); err != nil {
		return nil, fmt.Errorf("set input: %w", err)
	}

	result, err := runWithTimeout(ctx, vm, script+";\nmain(__input__);")
	if err != nil {
		return nil, err
	}

	exported := result.Export()
	if exported == nil {
		return proxies, nil
	}

	resultSlice, ok := exported.([]interface{})
	if !ok {
		return nil, fmt.Errorf("script must return an array, got %T", exported)
	}

	out := make([]map[string]interface{}, 0, len(resultSlice))
	for _, item := range resultSlice {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}

	return out, nil
}

func runWithTimeout(ctx context.Context, vm *goja.Runtime, code string) (goja.Value, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	vm.ClearInterrupt()

	go func() {
		<-timeoutCtx.Done()
		if timeoutCtx.Err() == context.DeadlineExceeded {
			vm.Interrupt("script execution timeout (5s)")
		}
	}()

	result, err := vm.RunString(code)
	if err != nil {
		if interrupted, ok := err.(*goja.InterruptedError); ok {
			return nil, fmt.Errorf("script interrupted: %s", interrupted.Value())
		}
		return nil, fmt.Errorf("script error: %w", err)
	}

	return result, nil
}
