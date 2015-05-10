package winny

import (
	"testing"
)

func TestDecryptNodeStr(t *testing.T) {
	inputs := []string{
		"@d5c84ca7c50a22896601dbc6924cef6bcd80e51b",
		"@2916b4b3466e63924623aeea022ebab33163ac7696e5",
		"@662f11b40e2940daf963f0a98755945fb595",
		"@3c1d0495e7dc45c3315df7701be6eddc2bf009b8b9fe",
		"@f730cabc6e05837cd87bca1352df0545adc8863b59527bcd10210d7154c205e0c51b2b71548d5bdac5"}
	expects := []string{
		"49.253.181.126:6566",
		"192.168.100.101:22892",
		"192.168.0.2:28173",
		"111.249.228.239:17884",
		"pl369.nas81a.p-ibaraki.nttpc.ne.jp:22739"}

	for i := 0; i < len(inputs); i++ {
		actual, err := DecryptNodeString(inputs[i])
		if err != nil {
			t.Errorf("error %v\ninput %#v\nexpect %#v\n", err, inputs[i], expects[i])
		}
		if actual != expects[i] {
			t.Errorf("input %#v\nexpect %#v\nactual %#v\n", inputs[i], expects[i], actual)
		}
		actualEncd, err := EncryptNodeString(actual)
		if err != nil {
			t.Errorf("error %v\nexpect %#v\n", err, inputs[i])
		}
		if actualEncd != inputs[i] {
			t.Errorf("expect %#v\nactual %#v\n", inputs[i], actualEncd)
		}
	}

}
