package migrate

import "testing"

func TestTestDBRequestValidate(t *testing.T) {
	valid := testDBRequest{
		Host: "127.0.0.1",
		Port: 5432,
		User: "postgres",
	}

	tests := []struct {
		name    string
		mutate  func(*testDBRequest)
		wantErr bool
	}{
		{name: "合法测试连接请求通过"},
		{name: "测试连接允许数据库名为空", mutate: func(r *testDBRequest) { r.Database = "" }},
		{name: "主机为空失败", mutate: func(r *testDBRequest) { r.Host = " " }, wantErr: true},
		{name: "端口为 0 失败", mutate: func(r *testDBRequest) { r.Port = 0 }, wantErr: true},
		{name: "端口超过上限失败", mutate: func(r *testDBRequest) { r.Port = 65536 }, wantErr: true},
		{name: "用户为空失败", mutate: func(r *testDBRequest) { r.User = " " }, wantErr: true},
		{name: "字段会去除首尾空白", mutate: func(r *testDBRequest) {
			r.Host = " 127.0.0.1 "
			r.User = " postgres "
			r.Database = " qms "
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			if tt.mutate != nil {
				tt.mutate(&req)
			}
			err := req.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.name == "字段会去除首尾空白" {
				if req.Host != "127.0.0.1" || req.User != "postgres" || req.Database != "qms" {
					t.Fatalf("trimmed fields = host %q user %q database %q", req.Host, req.User, req.Database)
				}
			}
		})
	}
}

func TestSaveConfigRequestValidate(t *testing.T) {
	valid := saveConfigRequest{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "postgres",
		Database: "qmediasync",
	}

	tests := []struct {
		name    string
		mutate  func(*saveConfigRequest)
		wantErr bool
	}{
		{name: "合法保存配置请求通过"},
		{name: "主机为空失败", mutate: func(r *saveConfigRequest) { r.Host = " " }, wantErr: true},
		{name: "端口为 0 失败", mutate: func(r *saveConfigRequest) { r.Port = 0 }, wantErr: true},
		{name: "端口超过上限失败", mutate: func(r *saveConfigRequest) { r.Port = 65536 }, wantErr: true},
		{name: "用户为空失败", mutate: func(r *saveConfigRequest) { r.User = " " }, wantErr: true},
		{name: "数据库名为空失败", mutate: func(r *saveConfigRequest) { r.Database = " " }, wantErr: true},
		{name: "字段会去除首尾空白", mutate: func(r *saveConfigRequest) {
			r.Host = " 127.0.0.1 "
			r.User = " postgres "
			r.Database = " qmediasync "
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			if tt.mutate != nil {
				tt.mutate(&req)
			}
			err := req.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.name == "字段会去除首尾空白" {
				if req.Host != "127.0.0.1" || req.User != "postgres" || req.Database != "qmediasync" {
					t.Fatalf("trimmed fields = host %q user %q database %q", req.Host, req.User, req.Database)
				}
			}
		})
	}
}
