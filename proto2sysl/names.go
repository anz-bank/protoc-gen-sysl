package proto2sysl

import (
	"strings"

	"github.com/anz-bank/protoc-gen-sysl/newsysl"
	"github.com/anz-bank/protoc-gen-sysl/sysl"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const typesAppName = "Types"

// names represents the set of relevant names to identify an element of a protobuf.
type names struct {
	// The protobuf package of the descriptor's file.
	protoPackage string
	// The Sysl namespace specified for the descriptor (directly or in the file).
	namespace []string
	// The names of the parent descriptors within which the descriptor is nested.
	parentNames []string
	// The name of the descriptor.
	name string
	// The name of the application corresponding to or containing the descriptor, including its
	// namespace.
	appName []string
	// The fully qualified name of the descriptor, including namespace and parents.
	fullName string
}

// getNames returns all the relevant names to identify a descriptor.
func getNames(ctx GenContext, d protoreflect.Descriptor) names {
	pkg := string(d.ParentFile().Package())
	parent := string(d.FullName().Parent())
	parents := []string{}
	if len(parent) > len(pkg) {
		parents = strings.Split(parent[len(pkg)+1:], ".")
	}
	for i, p := range parents {
		parents[i] = newsysl.SanitiseTypeName(p)
	}

	ns := getFileNamespaceOption(d.ParentFile())

	for i, n := range ns {
		ns[i] = newsysl.SanitiseTypeName(n)
	}
	name := newsysl.SanitiseTypeName(string(d.Name()))

	var appName []string
	var fullName string
	switch d.(type) {
	case protoreflect.ServiceDescriptor:
		appName = append(ns, name)
		fullName = namespaceJoin(appName)

	case protoreflect.MethodDescriptor:
		appName = append(ns, string(d.Parent().Name()))
		fullName = parentJoin(namespaceJoin(appName), name)

	case protoreflect.MessageDescriptor:
		// Check whether the message has been used and recorded by a service method.
		path := append(parents, name)
		for _, app := range ctx.module.Apps {
			if len(app.Name.Part) == len(ns)+1 &&
				namespaceJoin(app.Name.Part[:len(ns)]) == namespaceJoin(ns) {
				if _, ok := app.Types[path[0]]; ok {
					appName = app.Name.Part
					fullName = parentJoin(namespaceJoin(appName), parentJoin(path...))
					break
				}
			}
		}
	}
	if appName == nil {
		// If a Sysl namespace is provided, assume that's sufficient to disambiguate applications
		// such that we can use the app name "Types". Without a namespace, name the app after the
		// proto package to avoid collisions. Combining a namespace with a package name would most
		// likely produce unwieldy and redundant names.
		if len(ns) > 0 {
			appName = append(ns, typesAppName)
		} else {
			appName = []string{packageToApp(pkg)}
		}
		fullName = parentJoin(namespaceJoin(appName), parentJoin(append(parents, name)...))
	}

	return names{
		protoPackage: pkg,
		namespace:    ns,
		parentNames:  parents,
		name:         name,
		appName:      appName,
		fullName:     fullName,
	}
}

// namespaceJoin joins an array of names on the " :: " namespace separator.
func namespaceJoin(names []string) string {
	return strings.Join(names, " :: ")
}

// parentJoin joins an array of names on the "." parent separator.
func parentJoin(names ...string) string {
	return strings.Join(names, ".")
}

// packageToApp returns the name of the app representing an external protobuf package.
func packageToApp(pkg string) string {
	return strings.ReplaceAll(pkg, ".", "_")
}

// getFileNamespaceOption returns the value of the sysl.namespace FileOption, split on "::".
func getFileNamespaceOption(d protoreflect.FileDescriptor) []string {
	return getNameOption(d, sysl.E_Namespace)
}

// getNameOption returns the value of an option split and cleaned to resemble an app name, or an
// empty array if the given option is not present.
//
// Intend for the retrieval of namespace and app name values from options.
func getNameOption(d protoreflect.Descriptor, o protoreflect.ExtensionType) []string {
	opts := d.Options()
	if opts == nil {
		return []string{}
	}
	opt := proto.GetExtension(opts, o).(string)
	if opt == "" {
		return []string{}
	}

	namespace := strings.Split(opt, "::")
	for i, n := range namespace {
		namespace[i] = strings.TrimSpace(n)
	}
	return namespace
}
