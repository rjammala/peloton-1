// @generated AUTO GENERATED - DO NOT EDIT! 117d51fa2854b0184adc875246a35929bbbf0a91

// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package labels

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplate(t *testing.T) {
	template := NewTemplate("foo", "$bar$", "$baz$")
	label1 := template.Instantiate()

	assert.Equal(t, "foo.$bar$.$baz$", label1.String())

	template.Bind("bar", "bar")
	label2 := template.Instantiate()
	assert.Equal(t, "foo.bar.$baz$", label2.String())

	template.Bind("baz", "baz")
	label3 := template.Instantiate()
	assert.Equal(t, "foo.bar.baz", label3.String())
}

func TestTemplate_Mappings(t *testing.T) {
	template := NewTemplate("foo", "$bar$", "$baz$")
	template.Bind("bar", "bar")

	assert.Equal(t, map[string]string{"bar": "bar", "baz": ""}, template.Mappings())
}
