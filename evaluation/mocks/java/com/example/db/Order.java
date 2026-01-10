package com.example.db;

import java.util.List;

public class Order {
    private String id;
    private String userId;
    private List<String> items;
    private String status;
    private String paymentId;

    public Order(String id, String userId, List<String> items) {
        this.id = id;
        this.userId = userId;
        this.items = items;
        this.status = "PENDING";
    }

    // Getters and Setters
    public String getId() { return id; }
    public void setStatus(String status) { this.status = status; }
    public void setPaymentId(String paymentId) { this.paymentId = paymentId; }
}
