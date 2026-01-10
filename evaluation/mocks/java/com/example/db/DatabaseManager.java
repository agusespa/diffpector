package com.example.db;

import java.util.function.Consumer;

public class DatabaseManager {
    
    public void runInTransaction(Runnable action) {
        System.out.println("BEGIN TRANSACTION");
        try {
            action.run();
            System.out.println("COMMIT");
        } catch (Exception e) {
            System.out.println("ROLLBACK");
            throw e;
        }
    }

    public void createOrder(Order order) {
        // Simulate INSERT
        System.out.println("INSERT INTO orders " + order.getId());
    }

    public void updateOrderStatus(String orderId, String status, String paymentId) {
        // Simulate UPDATE
        System.out.println("UPDATE orders SET status=" + status + " WHERE id=" + orderId);
    }
}
