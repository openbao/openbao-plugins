package plugin

// func testConfigCreate(t *testing.T, b *azureAuthBackend, s logical.Storage, d map[string]interface{}) {
// 	resp, err := b.HandleRequest(context.Background(), &logical.Request{
// 		Operation: logical.CreateOperation,
// 		Path:      fmt.Sprintf("config"),
// 		Data:      d,
// 		Storage:   s,
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if resp != nil && resp.IsError() {
// 		t.Fatal(resp.Error())
// 	}
// 	b.oidcVerifier = newMockVerifier()
// }
