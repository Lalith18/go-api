
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "math"
    "net/http"
    "sync"
)

type Order struct {
    OrderID     string           `json:"orderId"`
    OrderDate   string           `json:"orderDate"`
    ProductList []OrderedProduct `json:"products"`
}

type OrderedProduct struct {
    ProductID string `json:"productId"`
    Quantity  int64  `json:"quantity"`
}

type OrderFulfillmentStatus struct {
    OrderID               string                 `json:"orderId"`
    OrderStatus           string                 `json:"orderStatus"`
    OrderFulfillmentDetails []OrderFulfillmentDetail `json:"orderDetails"`
    OrderTotalCost        float64                `json:"orderTotalCost"`
}

type OrderFulfillmentDetail struct {
    ProductID          string  `json:"productId"`
    SupplierID         string  `json:"supplierId"`
    QuantityFulfilled  int64   `json:"quantityFulfilled"`
    IndividualCost     float64 `json:"individualCost"`
}

type Product struct {
    ProductID   string
    SupplierIds []string
    ProductPrice float64
}

var staticProductDetails = map[string]Product{
    // This would typically come from an external source or be hardcoded as per the problem definition
    "P001": {ProductID: "P001", SupplierIds: []string{"S1", "S2", "S3"}, ProductPrice: 50},
    "P002": {ProductID: "P002", SupplierIds: []string{"S1", "S3"}, ProductPrice: 30},
}

// Function to handle parallel processing of each product
func getResponseForOrder(order Order) OrderFulfillmentStatus {
    var orderTotalCost float64
    orderResponse := OrderFulfillmentStatus{
        OrderID:              order.OrderID,
        OrderStatus:          "FULFILLED",
        OrderFulfillmentDetails: []OrderFulfillmentDetail{},
    }

    var wg sync.WaitGroup
    var mu sync.Mutex

    for i := 0; i < len(order.ProductList); i++ {
        orderDetail := order.ProductList[i]
        orderQuantityRequested := orderDetail.Quantity
        productID := orderDetail.ProductID

        suppliers := staticProductDetails[productID].SupplierIds

        wg.Add(1)
        go func(orderDetail OrderedProduct, suppliers []string) {
            defer wg.Done()

            var orderQuantityRemaining = int64(orderQuantityRequested)
            var orderFulfillmentDetails OrderFulfillmentDetail

            // Iterate over suppliers and fulfill up to 40% or max 100,000 units
            for j := 0; j < len(suppliers); j++ {
                if orderQuantityRemaining == 0 {
                    break
                }

                quantitySupplied := minInt(orderQuantityRemaining, minInt(100000, int64(float64(orderQuantityRequested)*40/100)))
                orderFulfillmentDetails = OrderFulfillmentDetail{
                    ProductID:         productID,
                    SupplierID:        suppliers[j],
                    QuantityFulfilled: quantitySupplied,
                    IndividualCost:    staticProductDetails[productID].ProductPrice * float64(quantitySupplied),
                }

                mu.Lock()
                orderTotalCost += orderFulfillmentDetails.IndividualCost
                orderResponse.OrderFulfillmentDetails = append(orderResponse.OrderFulfillmentDetails, orderFulfillmentDetails)
                mu.Unlock()

                orderQuantityRemaining -= quantitySupplied
            }

            if orderQuantityRemaining != 0 {
                mu.Lock()
                orderResponse.OrderStatus = "FAILED"
                mu.Unlock()
            }
        }(orderDetail, suppliers)
    }

    wg.Wait()
    orderResponse.OrderTotalCost = orderTotalCost
    fmt.Printf("Successfully processed order id: %s\n", order.OrderID)
    return orderResponse
}

func minInt(a, b int64) int64 {
    if a < b {
        return a
    }
    return b
}

func httpFunc(w http.ResponseWriter, r *http.Request) {
    var request Order
    err := json.NewDecoder(r.Body).Decode(&request)
    if err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    var response = getResponseForOrder(request)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    http.HandleFunc("/retail-store/v1/order", httpFunc)
    log.Fatal(http.ListenAndServe(":9090", nil))
}
