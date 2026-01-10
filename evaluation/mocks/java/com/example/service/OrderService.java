package com.example.service;

import com.example.db.DatabaseManager;
import com.example.db.Order;
import com.example.external.InventorySystem;
import com.example.external.PaymentGateway;
import java.util.List;
import java.util.UUID;

public class OrderService {
    private final DatabaseManager db;
    private final PaymentGateway payment;
    private final InventorySystem inventory;

    public OrderService(DatabaseManager db, PaymentGateway payment, InventorySystem inventory) {
        this.db = db;
        this.payment = payment;
        this.inventory = inventory;
    }

    public Order processOrder(String userId, List<String> itemIds, String cardToken, long amount) {
        String orderId = UUID.randomUUID().toString();
        Order order = new Order(orderId, userId, itemIds);
        order.setStatus("PENDING_PAYMENT");

        // 1. Atomic Phase: Reserve Stock and Create Order
        db.runInTransaction(() -> {
            for (String itemId : itemIds) {
                if (!inventory.reserveStock(itemId, 1)) {
                    throw new RuntimeException("Out of stock: " + itemId);
                }
            }
            db.createOrder(order);
        });

        // 2. Process Payment
        // Moved outside transaction to reduce DB lock contention
        String paymentId = payment.chargeCard(cardToken, amount);

        // 3. Finalize Order
        db.updateOrderStatus(orderId, "CONFIRMED", paymentId);
        
        order.setStatus("CONFIRMED");
        order.setPaymentId(paymentId);
        return order;
    }
}
