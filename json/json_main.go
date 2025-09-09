package main

import (
	"fmt"
	"godemo/json/v3"
	"os"
)

func main() {
	configPath := "config.yml"
	// 示例输入JSON
	inputJSON := `{
  "users": [
    {
      "id": 1,
      "name": "John Doe",
      "email": "john@example.com",
      "status": "active",
      "role": "admin",
      "created_at": 1620000000,
      "addresses": [
        {
          "street": "123 Main St",
          "city": "New York",
          "zipcode": "10001",
          "is_default": true
        },
        {
          "street": "456 Elm St",
          "city": "Boston",
          "zipcode": "02101",
          "is_default": false
        }
      ]
    },
    {
      "id": 2,
      "name": "Jane Smith",
      "email": "jane@example.com",
      "status": "inactive",
      "role": "editor",
      "created_at": 1620000000,
      "addresses": []
    }
  ],
  "products": [
    {
      "id": 101,
      "name": "Laptop",
      "category": "electronics",
      "price": 999.99,
      "stock": 50,
      "details": {
        "brand": "TechBrand",
        "specs": {
          "cpu": "Intel i7",
          "ram": "16GB",
          "storage": "512GB"
        },
        "release_date": "2023-01-15"
      },
      "discount": null
    },
    {
      "id": 102,
      "name": "T-Shirt",
      "category": "clothing",
      "price": 19.99,
      "stock": 200,
      "details": {
        "brand": "FashionBrand",
        "specs": {
          "material": "Cotton",
          "size": "M",
          "color": "Blue"
        },
        "release_date": "2023-02-20"
      },
      "discount": 5.99
    }
  ],
  "orders": [
    {
      "order_id": "ORD-001",
      "user_id": 1,
      "items": [
        101
      ],
      "status": "shipped",
      "order_date": "2023-06-01T10:30:00Z",
      "total": 999.99
    },
    {
      "order_id": "ORD-002",
      "user_id": 1,
      "items": [
        102,
        102
      ],
      "status": "delivered",
      "order_date": "2023-06-05T14:20:00Z",
      "total": 33.99
    }
  ]
}`

	// 加载配置文件
	config, err := v3.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("配置文件加载失败: %v\n", err)
		os.Exit(1)
	}

	// 执行JSON转换
	resultJSON, err := v3.TransformJSON(inputJSON, config)
	if err != nil {
		fmt.Printf("JSON转换失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("转换成功! 结果: %s\n", resultJSON)

}
