package govfx

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
)

//==============================================================================

// GetComputedStyle returns the dom.Element computed css styles.
func GetComputedStyle(elem dom.Element, ps string) (*dom.CSSStyleDeclaration, error) {
	css := Window().GetComputedStyle(elem, ps)
	if css == nil {
		return nil, ErrNotFound
	}

	return css, nil
}

// RemoveComputedStyleValue removes the value of the property from the computed
// style object.
func RemoveComputedStyleValue(css *dom.CSSStyleDeclaration, prop string) {
	defer func() {
		recover()
	}()

	css.Call("removeProperty", prop)
}

// GetComputedStyleValue retrieves the value of the property from the computed
// style object.
func GetComputedStyleValue(elem dom.Element, psudo string, prop string) (*js.Object, error) {
	vs, err := GetComputedStyle(elem, psudo)
	if err != nil {
		return nil, err
	}

	vcs, err := GetComputedStyleValueWith(vs, prop)
	if err != nil {
		return nil, err
	}

	return vcs, nil
}

// GetComputedStyleValueWith usings the CSSStyleDeclaration to
// retrieves the value of the property from the computed
// style object.
func GetComputedStyleValueWith(css *dom.CSSStyleDeclaration, prop string) (*js.Object, error) {
	vs := css.Call("getPropertyValue", prop)
	if vs == nil {
		return nil, ErrNotFound
	}

	return vs, nil
}

// GetComputedStylePriority retrieves the proritiy of the property from the computed
// style object.
func GetComputedStylePriority(css *dom.CSSStyleDeclaration, prop string) (int, error) {
	vs := css.Call("getPropertyPriority", prop)
	if vs == nil {
		return 0, ErrNotFound
	}

	if strings.TrimSpace(vs.String()) == "" {
		return 0, nil
	}

	return 1, nil
}

//==============================================================================

// ComputedStyle defines a style property item.
type ComputedStyle struct {
	Name       string
	VendorName string
	Value      string
	Values     []string
	Priority   bool // values between [0,1] to indicate use of '!important'
}

// ComputedStyleMap defines a map type of computed style properties and values.
type ComputedStyleMap map[string]*ComputedStyle

// GetComputedStyleMap returns a map of computed style properties and values.
// Also all vendored names are cleaned up to allow quick and easy access
// regardless of vendor.
func GetComputedStyleMap(elem dom.Element, ps string) (ComputedStyleMap, error) {
	css, err := GetComputedStyle(elem, ps)
	if err != nil {
		return nil, err
	}

	styleMap := make(ComputedStyleMap)

	// Get the map and pull the necessary property:value and importance facts.
	for key, val := range css.ToMap() {
		priority, _ := GetComputedStylePriority(css, key)

		unvendoredName := key

		// Clean key of any vendored name to allow easy access.
		for _, vo := range vendorTags {
			unvendoredName = strings.TrimPrefix(unvendoredName, fmt.Sprintf("%s", vo))
			unvendoredName = strings.TrimPrefix(unvendoredName, fmt.Sprintf("-%s-", vo))
		}

		var vals []string

		if strings.TrimSpace(val) != "none" {
			vals = append(vals, val)
		}

		styleMap[unvendoredName] = &ComputedStyle{
			Name:       unvendoredName,
			VendorName: key,
			Value:      val,
			Values:     vals,
			Priority:   (priority > 0),
		}
	}

	return styleMap, nil
}

// Add adjusts the stylemap with a new property.
func (c ComputedStyleMap) Add(name string, value string, priority bool) {
	if !c.Has(name) {
		c[name] = &ComputedStyle{
			Name:       name,
			VendorName: name,
			Value:      value,
			Values:     []string{value},
			Priority:   priority,
		}
		return
	}

	m := c[name]
	m.Value = value
	m.Priority = priority
	m.Values = []string{value}
}

// propName defines a regexp to pull the name of a css property setter.
var propName = regexp.MustCompile("([\\w\\-0-9]+)\\(?\\)?")

// AddMore adjusts the stylemap for a exisiting property adding the new value
// into the values lists else adds as just a new value if it does not exists.
func (c ComputedStyleMap) AddMore(name string, value string, priority bool) {
	if !c.Has(name) {
		c[name] = &ComputedStyle{
			Name:       name,
			VendorName: name,
			Value:      value,
			Values:     []string{value},
			Priority:   priority,
		}
		return
	}

	m := c[name]

	if len(m.Values) == 1 && m.Values[0] == "none" {
		m.Values[0] = value
		return
	}

	var found bool

	prop := propName.FindStringSubmatch(value)[1]

	// We must first check if we have a property with the name in list then
	// replace it else append it into list.
	for ind, val := range m.Values {

		valName := propName.FindStringSubmatch(val)[1]

		if valName != prop {
			continue
		}

		m.Values[ind] = value
		found = true
	}

	// If not found, then append
	if !found {
		m.Values = append(m.Values, value)
	}
}

// Has returns true/false if the property exists.
func (c ComputedStyleMap) Has(name string) bool {
	_, ok := c[name]
	return ok
}

// Get retrieves the specific property if it exists.
func (c ComputedStyleMap) Get(name string) (*ComputedStyle, error) {
	cs, ok := c[name]

	if !ok {
		return nil, ErrNotFound
	}

	return cs, nil
}

//==============================================================================

// ToRGB turns a hexademicmal color into rgba format.
// Returns the read, green and blue values as int.
func ToRGB(hex string) (red, green, blue int) {
	if strings.HasPrefix(hex, "#") {
		hex = strings.TrimPrefix(hex, "#")
	}

	// We are dealing with a 3 string hex.
	if len(hex) < 6 {
		parts := strings.Split(hex, "")
		red = parseIntBase16(doubleString(parts[0]))
		green = parseIntBase16(doubleString(parts[1]))
		blue = parseIntBase16(doubleString(parts[2]))
		return
	}

	red = parseIntBase16(hex[0:2])
	green = parseIntBase16(hex[2:4])
	blue = parseIntBase16(hex[4:6])

	return
}

// RGBA turns a hexademicmal color into rgba format.
// Alpha values ranges from 0-100
func RGBA(hex string, alpha int) string {
	r, g, b := ToRGB(hex)
	return fmt.Sprintf("rgba(%d,%d,%d,%.2f)", r, g, b, float64(alpha)/100)
}

// vendorTags provides a lists of different browser specific vendor names.
var vendorTags = []string{"moz", "webki", "O", "ms"}

// Vendorize returns a property name with the different versions known according
// browsers.
func Vendorize(u string) []string {
	var v []string

	for _, vn := range vendorTags {
		v = append(v, fmt.Sprintf("-%s-%s", vn, u))
	}

	return v
}

// Unit returns a valid unit type in the browser, if the supplied unit is
// standard then it is return else 'px' is returned as default.
func Unit(u string) string {
	switch u {
	case "rem":
		return u
	case "em":
		return u
	case "px":
		return u
	case "%":
		return u
	case "vw":
		return u
	default:
		return "px"
	}
}

// doubleString doubles the giving string.
func doubleString(c string) string {
	return fmt.Sprintf("%s%s", c, c)
}

//==============================================================================
