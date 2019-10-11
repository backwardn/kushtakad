package clone

import (
	"net/url"
	"testing"
)

var cases = []struct {
	parent string
	uri    string
	want   string
}{
	{
		"http://www.example.com",
		"http://example.com/UserFiles/Servers/Server_543/Templates/2017/img/map-bg.png",
		"http://example.com/UserFiles/Servers/Server_543/Templates/2017/img/map-bg.png",
	},
	{
		"http://www.example.com/a/b/c/d/e/f/g/",
		"../../../../updates/concrete5.7.5.6/concrete/../concrete/images/devices/galaxy/s5-landscape.png",
		"http://www.example.com/a/b/c/updates/concrete5.7.5.6/concrete/images/devices/galaxy/s5-landscape.png",
	},
	{
		"http://www.example.com/a/b/c/d/e/f/g/",
		"/application/static/blps/bootstrap/app/fonts/glyphicons-halflings-regular.eot?#iefix",
		"http://www.example.com/application/static/blps/bootstrap/app/fonts/glyphicons-halflings-regular.eot?#iefix",
	},
	{
		"http://www.example.com/a/b/c/d/e/f/g/file.css",
		"./application/static/blps/bootstrap/app/fonts/glyphicons-halflings-regular.eot?#iefix",
		"http://www.example.com/a/b/c/d/e/f/g/application/static/blps/bootstrap/app/fonts/glyphicons-halflings-regular.eot?#iefix",
	},
	{
		"http://www.example.com/a/b/c/d/e/f/g/file.css",
		"https://maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css",
		"https://maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css",
	},
	{
		"https://www.example.z12.zz.us/application/files/cache/css/172_style.css",
		"/application/static/blps/bootstrap/app/fonts/glyphicons-halflings-regular.ttf",
		"https://www.example.z12.zz.us/application/static/blps/bootstrap/app/fonts/glyphicons-halflings-regular.ttf",
	},
	{
		"http://www.example.com/a/b/c/d/e/f/g/file.css",
		"https://maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css",
		"https://maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css",
	},
	{
		"https://www.example.org/common/pages/Portalstatictyle.ashx?templateId=2739",
		"//example.org/UserFiles/Servers/Server_543/Templates/2017/img/map-bg.png",
		"https://example.org/UserFiles/Servers/Server_543/Templates/2017/img/map-bg.png",
	},
	{
		"https://www.example.net/Static//site/assets/styles/dashboard.css",
		"../../../GlobalAssets/webfonts/NotoSerif-Regular.ttf",
		"https://www.example.net/Static//GlobalAssets/webfonts/NotoSerif-Regular.ttf",
	},
}

func TestFunc(t *testing.T) {
	for _, c := range cases {
		puri, err := url.Parse(c.parent)
		if err != nil {
			t.Fatal(err)
		}

		got, err := absUrl(c.uri, puri)
		if err != nil {
			t.Fatal(err)
		}

		if got.String() != c.want {
			t.Errorf("absUrl(%q) == %q, want %q", c.uri, got, c.want)
		}
	}
}
