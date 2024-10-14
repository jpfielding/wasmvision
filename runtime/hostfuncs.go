package runtime

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/orsinium-labs/wypes"
	"github.com/wasmvision/wasmvision/cv"
)

func hostedModules(ctx *cv.Context) wypes.Modules {
	return wypes.Modules{
		"wasmvision:platform/config": wypes.Module{
			"get-config": wypes.H3(hostGetConfigFunc(ctx)),
		},
		"wasmvision:platform/http": wypes.Module{
			"get":  wypes.H3(httpGetFunc(ctx)),
			"post": wypes.H5(httpPostFunc(ctx)),
		},
		"wasmvision:platform/logging": wypes.Module{
			"println": wypes.H1(hostPrintln),
			"log":     wypes.H1(hostLogFunc(ctx)),
		},
		"wasmvision:platform/time": wypes.Module{
			"now": wypes.H2(hostTimeNowFunc(ctx)),
		},
	}
}

func hostPrintln(msg wypes.String) wypes.Void {
	fmt.Println(msg.Unwrap())
	return wypes.Void{}
}

func hostLogFunc(ctx *cv.Context) func(wypes.String) wypes.Void {
	return func(msg wypes.String) wypes.Void {
		if ctx.Logging {
			log.Println(msg.Unwrap())
		}
		return wypes.Void{}
	}
}

func hostGetConfigFunc(ctx *cv.Context) func(*wypes.Store, wypes.String, wypes.Result[wypes.String, wypes.String, wypes.UInt32]) wypes.Void {
	return func(s *wypes.Store, key wypes.String, result wypes.Result[wypes.String, wypes.String, wypes.UInt32]) wypes.Void {
		if s.Error != nil {
			log.Printf("error in store after lift: %v\n", s.Error)
		}
		val, ok := ctx.Config.Get(key.Unwrap())
		if !ok {
			result.IsError = true
			result.Error = wypes.UInt32(1) // no-such-key
		} else {
			result.IsError = false
			result.OK = wypes.String{Raw: val}
		}

		result.DataPtr = ctx.ReturnDataPtr
		result.Lower(s)

		if s.Error != nil {
			log.Printf("error in store after lower: %v\n", s.Error)
		}

		return wypes.Void{}
	}
}

func httpGetFunc(ctx *cv.Context) func(*wypes.Store, wypes.String, wypes.Result[wypes.Bytes, wypes.Bytes, wypes.UInt32]) wypes.Void {
	return func(s *wypes.Store, url wypes.String, result wypes.Result[wypes.Bytes, wypes.Bytes, wypes.UInt32]) wypes.Void {
		if ctx.Logging {
			log.Println("http get:", url.Unwrap())
		}

		resp, err := http.Get(url.Unwrap())
		if err != nil {
			result.IsError = true
			result.Error = wypes.UInt32(3) // HTTPErrorRequestError
			return wypes.Void{}
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			result.IsError = true
			result.Error = wypes.UInt32(3) // HTTPErrorRequestError
			return wypes.Void{}
		}

		// HACK: limit the size of the response to 128 bytes, for now.
		max := 128
		if len(body) < max {
			max = len(body)
		}

		result.IsError = false
		result.OK = wypes.Bytes{Raw: body[:max]}
		result.DataPtr = ctx.ReturnDataPtr

		result.Lower(s)
		if s.Error != nil {
			log.Printf("httpGetFunc error in store after lower: %v\n", s.Error)
		}

		return wypes.Void{}
	}
}

func httpPostFunc(ctx *cv.Context) func(*wypes.Store, wypes.String, wypes.String, wypes.Bytes, wypes.Result[wypes.Bytes, wypes.Bytes, wypes.UInt32]) wypes.Void {
	return func(s *wypes.Store, url wypes.String, contentType wypes.String, request wypes.Bytes, result wypes.Result[wypes.Bytes, wypes.Bytes, wypes.UInt32]) wypes.Void {
		if ctx.Logging {
			log.Println("http post:", url.Unwrap())
		}

		reqBody := bytes.NewBuffer(request.Raw)

		resp, err := http.Post(url.Unwrap(), contentType.Unwrap(), reqBody)
		if err != nil {
			result.IsError = true
			result.Error = wypes.UInt32(3) // HTTPErrorRequestError
			return wypes.Void{}
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			result.IsError = true
			result.Error = wypes.UInt32(3) // HTTPErrorRequestError
			return wypes.Void{}
		}

		// HACK: limit the size of the response to 128 bytes, for now.
		max := 128
		if len(body) < max {
			max = len(body)
		}

		result.IsError = false
		result.OK = wypes.Bytes{Raw: body[:max]}
		result.DataPtr = ctx.ReturnDataPtr

		result.Lower(s)
		if s.Error != nil {
			log.Printf("httpGetFunc error in store after lower: %v\n", s.Error)
		}

		return wypes.Void{}
	}
}

func hostTimeNowFunc(ctx *cv.Context) func(*wypes.Store, wypes.UInt32) wypes.UInt64 {
	return func(*wypes.Store, wypes.UInt32) wypes.UInt64 {
		t := time.Now().UnixMicro()

		return wypes.UInt64(t)
	}
}
