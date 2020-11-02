package flag

import (
	"fmt"
	"os"
)

func (f Flag) Remove() {
	if f.Email == "" {
		fmt.Println("请设置 -email 参数来选择要删除的保存的账号")
		os.Exit(0)
	}
	if _, ok := gmlconfig[f.ApiAddress][f.Email]; !ok {
		fmt.Println(f.ApiAddress, f.Email, "不存在")
	} else {
		delete(gmlconfig[f.ApiAddress], f.Email)
		fmt.Println("成功删除", f.ApiAddress, f.Email)
		if len(gmlconfig[f.ApiAddress]) == 0 {
			delete(gmlconfig, f.ApiAddress)
		}
		saveconfig()
	}
	os.Exit(0)
}