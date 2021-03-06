package compiler

import (
	"bytes"
	"fmt"
	"github.com/iancoleman/strcase"
	"log"
	"strings"
)

func getArgumentDataType(argument *endpointArgumentDefinition) string {
	return strcase.ToCamel(argument.UnitName)
}

func writeArgumentDefinition(w *bytes.Buffer, argument *endpointArgumentDefinition) {
	if argument.UnitName == "" {
		return
	}

	unitName := getArgumentDataType(argument)
	if argument.Streamed {
		_writelni(w, 1, fmt.Sprintf("Data chan *%s", unitName))
	} else {
		_writelni(w, 1, fmt.Sprintf("Data *%s", unitName))
	}
}

func writeRequest(w *bytes.Buffer, publicPathName string, includeData bool) {
	_writelni(w, 2, fmt.Sprintf("request := %s_Request{", publicPathName))
	if includeData {
		_writelni(w, 3, "Data: d,")
	}
	_writelni(w, 3, "Context: req.Context,")
	_writelni(w, 3, "Headers: req.Headers,")
	_writelni(w, 3, "Auth: req.Auth,")
	_writelni(w, 2, "}")
}

func writeHandlerCall(w *bytes.Buffer, errIsNew bool, doneCall bool, indentModifier int) {
	var colon string
	if errIsNew {
		colon = ":"
	}

	_writelni(w, 2+indentModifier, fmt.Sprintf("err %s= handler(&request, &response)", colon))
	_writelni(w, 2+indentModifier, "if err != nil {")
	_writelni(w, 3+indentModifier, "res.SendError(err)")
	_writelni(w, 2+indentModifier, "}")

	if doneCall {
		_writelni(w, 2+indentModifier, "done <- struct{}{}")
	}
}

func writeDecoderCall(w *bytes.Buffer, in, out string, inArgument *endpointArgumentDefinition, indentModifier int) {
	_writelni(w, 2+indentModifier, fmt.Sprintf("%s, err := Decode%s(%s)", out, getArgumentDataType(inArgument), in))
	_writelni(w, 2+indentModifier, "if err != nil {")
	_writelni(w, 3+indentModifier, "res.SendError(fmt.Errorf(\"handler for endpoint with ID %d failed to decode incoming message: %s\", endpointId, err.Error()))")
	_writelni(w, 3+indentModifier, "return")
	_writelni(w, 2+indentModifier, "}")
}

func writeClientStub(w *bytes.Buffer, endpointId uint16, publicPathName string, inArgument, outArgument endpointArgumentDefinition) {
	var (
		inName  = "encoder.UserFacingHermodUnit"
		outName = "client.DummyOutSample"
	)

	if inArgument.UnitName != "" {
		inName = getArgumentDataType(&inArgument)
	}

	if outArgument.UnitName != "" {
		outName = getArgumentDataType(&outArgument)
	}

	serviceReadWriterType := fmt.Sprintf(`client.ServiceReadWriter[%s, %s]`, inName, outName)
	_writeln(w, fmt.Sprintf("func Request%s(router *client.WebSocketRouter, token ...string) (*%s, error) {", publicPathName, serviceReadWriterType))
	_writelni(w, 1, fmt.Sprintf("rw := %s{", serviceReadWriterType))
	_writelni(w, 2, "Router: router,")
	_writelni(w, 2, fmt.Sprintf("Endpoint: %d,", endpointId))
	_writelni(w, 2, fmt.Sprintf("HasIn: %t,", inArgument.UnitName != ""))
	_writelni(w, 2, fmt.Sprintf("OutSample: %s{},", outName))
	_writelni(w, 1, "}")

	_writelni(w, 1, "err := rw.Init(token...)")
	_writelni(w, 1, "return &rw, err")
	_writeln(w, "}")
}

func writeService(w *bytes.Buffer, service *serviceDefinition, packageName string) (imports []string) {
	for _, endpoint := range service.Endpoints {
		publicPathComponents := strings.Split(endpoint.Path, "/")
		var publicPathName string
		for _, component := range publicPathComponents {
			publicPathName = strcase.ToCamel(component) + publicPathName
		}

		if endpoint.Id == 0xFFFF {
			log.Fatalln("encountered endpoint with illegal ID 0xFFFF")
			return
		}

		imports = append(imports, fmt.Sprintf("%s/client", packageName))
		writeClientStub(w, endpoint.Id, publicPathName, endpoint.In, endpoint.Out)

		_writeln(w, fmt.Sprintf("type %s_Request struct {", publicPathName))
		writeArgumentDefinition(w, &endpoint.In)

		imports = append(imports, "context")
		imports = append(imports, "net/http")
		imports = append(imports, fmt.Sprintf("%s/service", packageName))
		_writelni(w, 1, "Context context.Context")
		_writelni(w, 1, "Headers http.Header")
		_writelni(w, 1, "Auth *service.AuthAPI")
		_writeln(w, fmt.Sprintf("}"))

		_writeln(w, fmt.Sprintf("type %s_Response struct {", publicPathName))
		_writelni(w, 1, "sendFunction func(data *[]byte)")
		_writeln(w, "}")

		if endpoint.Out.UnitName != "" {
			_writeln(w, fmt.Sprintf("func (res *%s_Response) Send(data *%s) {", publicPathName, getArgumentDataType(&endpoint.Out)))
			_writelni(w, 1, "encoded, err := data.Encode()")
			_writelni(w, 1, "if err != nil {")
			_writelni(w, 2, "t := []byte(\"couldn't encode data\")")
			_writelni(w, 2, "res.sendFunction(&t)")
			_writelni(w, 2, "return")
			_writelni(w, 1, "}")
			_writelni(w, 1, "res.sendFunction(encoded)")
			_writeln(w, "}")
		}

		_writeln(w, fmt.Sprintf("func Register%sHandler(handler func(req *%s_Request, res *%s_Response) error) {", publicPathName, publicPathName, publicPathName))
		_writelni(w, 1, fmt.Sprintf("endpointId := uint16(%d)", endpoint.Id))
		_writelni(w, 1, "service.RegisterEndpoint(endpointId, func(req *service.Request, res *service.Response) {")
		_writelni(w, 2, fmt.Sprintf("response := %s_Response{", publicPathName))
		_writelni(w, 3, "sendFunction: res.Send,")
		_writelni(w, 2, "}")

		if endpoint.In.UnitName != "" {
			imports = append(imports, "fmt")
			if endpoint.In.Streamed {
				_writelni(w, 2, fmt.Sprintf("d := make(chan *%s)", getArgumentDataType(&endpoint.In)))
				writeRequest(w, publicPathName, true)

				_writelni(w, 2, "done := make(chan struct{})")
				_writelni(w, 2, "go func() {")
				writeHandlerCall(w, true, true, 1)
				_writelni(w, 2, "}()")

				_writelni(w, 2, "for {")
				_writelni(w, 3, "select {")
				_writelni(w, 3, "case <-req.Context.Done():")
				_writelni(w, 4, "return")
				_writelni(w, 3, "case <-done:")
				_writelni(w, 4, "return")
				_writelni(w, 3, "case data := <-req.Data:")

				_writelni(w, 4, "if data == nil {")
				_writelni(w, 5, "continue")
				_writelni(w, 4, "}")

				writeDecoderCall(w, "data", "decoded", &endpoint.In, 2)
				_writelni(w, 4, "request.Data <- decoded")
				_writelni(w, 3, "}")
				_writelni(w, 2, "}")
			} else {
				_writelni(w, 2, "initialData, ok := <-req.Data")
				_writelni(w, 2, "if !ok {")
				_writelni(w, 3, "return")
				_writelni(w, 2, "}")

				writeDecoderCall(w, "initialData", "d", &endpoint.In, 0)

				writeRequest(w, publicPathName, true)
				writeHandlerCall(w, false, false, 0)
			}
		} else {
			writeRequest(w, publicPathName, false)
			writeHandlerCall(w, true, false, 0)
		}

		_writelni(w, 1, "})")
		_writeln(w, "}")
	}

	return
}
