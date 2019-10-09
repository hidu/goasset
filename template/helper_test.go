package template

import (
	"strings"
	"testing"
)

type helperCaseItem struct {
	Input  string
	Expect string
}

func (hc *helperCaseItem) CheckExpectEq(t *testing.T, result string) {
	expectTrim := strings.TrimSpace(hc.Expect)
	resultTrim := strings.TrimSpace(result)
	if expectTrim != resultTrim {
		t.Errorf("result wrong, not eq, expect=[%s], real=[%s],input=[%s]", expectTrim, result, hc.Input)
	}
}

func TestAssetRemove(t *testing.T) {

	cases := []*helperCaseItem{
		{
			Input: `//asset_remove_start()
package template
//asset_remove_end( )`,
			Expect: "",
		},
		{
			Input: `//asset_remove_start()
package template
//asset_remove_end( )
`,
			Expect: "",
		},
		{
			Input: `// asset_remove_start()
package template
// asset_remove_end()
`,
			Expect: "",
		},
		{
			Input: `//asset_remove_start()
package template
//asset_remove_end( )
a
//asset_remove_start()
package template
// asset_remove_end( )
b
`,
			Expect: `
a

b`,
		},
		{
			Input: `//asset_remove_start()package template
//asset_remove_end()`,
			Expect: "",
		},
		{
			Input:  `//asset_remove_start()package template//asset_remove_end()`,
			Expect: "",
		},
		{
			Input:  `//asset_remove_start() //asset_remove_end( )`,
			Expect: "",
		},
		{
			Input:  `//asset_remove_start()//asset_remove_end( )`,
			Expect: "",
		},
		{
			Input:  `//asset_remove_start()a//asset_remove_end() c //asset_remove_start()b//asset_remove_end() d`,
			Expect: "c  d",
		},
	}
	helper := newAssetHelper()
	for idx, caseItem := range cases {
		t.Logf("now run case(%d)", idx)
		bf, err := helper.Remove("", []byte(caseItem.Input))
		if err != nil {
			t.Errorf("case with error: %s", err.Error())
		}
		caseItem.CheckExpectEq(t, string(bf))
	}
}

func TestAssetInclude(t *testing.T) {
	t.Logf("AssetInclude")
	cases := []*helperCaseItem{
		{
			Input:  `//asset_include(a.txt)`,
			Expect: "hello",
		},
		{
			Input:  `// asset_include(a.txt)`,
			Expect: "hello",
		},
	}
	helper := newAssetHelper()
	for idx, caseItem := range cases {
		t.Logf("now run case(%d)", idx)
		bf, err := helper.Include("../testdata/b.txt", []byte(caseItem.Input))
		if err != nil {
			t.Errorf("case with error: %s", err.Error())
		}
		caseItem.CheckExpectEq(t, string(bf))
	}

	// 测试循环include
	_, err := helper.Include("../testdata/b.txt", []byte("// asset_include(b_1.txt )"))
	if err == nil {
		t.Errorf("expect has error")
	}
}

func TestAssetRemoveAbove(t *testing.T) {
	t.Logf("AssetRemoveAbove")
	cases := []*helperCaseItem{
		{
			Input: `asset
//asset_remove_above()`,
			Expect: "",
		},
		{
			Input: `
//asset_remove_above()
asset
`,
			Expect: "asset",
		},
	}

	helper := newAssetHelper()

	for idx, caseItem := range cases {
		t.Logf("now run case(%d)", idx)
		bf, err := helper.RemoveAbove("../testdata/b.txt", []byte(caseItem.Input))
		if err != nil {
			t.Errorf("case with error: %s", err.Error())
		}
		caseItem.CheckExpectEq(t, string(bf))
	}
}
