
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "sync"
)

// OrderFulfillmentStatus represents the fulfillment status of an order
type OrderFulfillmentStatus struct {
    OrderID                 string
    OrderStatus             string
    OrderFulfillmentDetails []OrderFulfillmentDetail
    OrderTotalCost          float64
}

// OrderFulfillmentDetail represents the details of the order fulfillment for a product
type OrderFulfillmentDetail struct {
    SupplierID        string
    ProductID         string
    QuantityFulfilled int
    OrderCost         float64
}

// Order represents an order request from the client
type Order struct {
    OrderID      string
    OrderDetails []OrderDetail
}

// OrderDetail represents details of an ordered product
type OrderDetail struct {
    ProductID string
    Quantity  int
}

// Static data to simulate products and suppliers
var staticProductDetails = map[string]Product{
    "P1": {SupplierIds: []string{"S1", "S2"}, ProductPrice: 10},
    "P2": {SupplierIds: []string{"S3", "S4"}, ProductPrice: 20},
}

// Product represents product details with supplier information
type Product struct {
    SupplierIds  []string
    ProductPrice float64
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
    if a < b {
        return a
    }
    return b
}

// getResponseForOrder processes an order and returns the fulfillment status
func getResponseForOrder(order Order, wg *sync.WaitGroup, resultChannel chan<- OrderFulfillmentStatus) {
    defer wg.Done()

    var orderTotalCost float64
    var orderResponse = OrderFulfillmentStatus{
        OrderID:     order.OrderID,
        OrderStatus: "FULFILLED",
    }

    // Process each product in the order
    for _, orderDetail := range order.OrderDetails {
        productID := orderDetail.ProductID
        suppliers := staticProductDetails[productID].SupplierIds // Get suppliers for the product

        var orderQuantityRemaining = orderDetail.Quantity

        // Loop through suppliers and fulfill the order
        for _, supplier := range suppliers {
            if orderQuantityRemaining == 0 {
                break
            }
            // Assume this is where you calculate how much a supplier can fulfill
            orderQuantityFulfilled := minInt(orderQuantityRemaining, 100000) // Mock fulfillment
            orderQuantityRemaining -= orderQuantityFulfilled

            // Accumulate order details and costs
            orderFulfillmentDetail := OrderFulfillmentDetail{
                SupplierID:        supplier,
                QuantityFulfilled: orderQuantityFulfilled,
                ProductID:         productID,
                OrderCost:         float64(orderQuantityFulfilled) * staticProductDetails[productID].ProductPrice,
            }
            orderResponse.OrderFulfillmentDetails = append(orderResponse.OrderFulfillmentDetails, orderFulfillmentDetail)
            orderTotalCost += orderFulfillmentDetail.OrderCost
        }

        // If order cannot be fully fulfilled, set status to "FAILED"
        if orderQuantityRemaining != 0 {
            orderResponse.OrderStatus = "FAILED"
            break // Stop further processing since the order has already failed
        }
    }

    orderResponse.OrderTotalCost = orderTotalCost
    resultChannel <- orderResponse
}

// httpFunc is the HTTP handler for processing order requests
func httpFunc(w http.ResponseWriter, r *http.Request) {
    var request Order
    err := json.NewDecoder(r.Body).Decode(&request)
    if err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Process the order concurrently
    resultChannel := make(chan OrderFulfillmentStatus)
    var wg sync.WaitGroup
    wg.Add(1)
    go getResponseForOrder(request, &wg, resultChannel)

    go func() {
        wg.Wait()
        close(resultChannel)
    }()

    // Wait for the result and send the response
    response := <-resultChannel
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    http.HandleFunc("/retail-store/v1/order", httpFunc)
    log.Fatal(http.ListenAndServe(":9090", nil))
}
