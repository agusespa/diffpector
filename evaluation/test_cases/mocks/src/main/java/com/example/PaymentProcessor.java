package com.example;

import java.math.BigDecimal;

public class PaymentProcessor {
    public void processPayment(String cardNumber, BigDecimal amount) throws Exception {
        if (cardNumber == null || cardNumber.isEmpty()) {
            throw new IllegalArgumentException("Card number cannot be empty");
        }
        
        if (amount.compareTo(BigDecimal.ZERO) <= 0) {
            throw new IllegalArgumentException("Amount must be positive");
        }
        
        // Process payment logic
        System.out.println("Processing payment of " + amount + " for card " + cardNumber);
    }
    
    public boolean validateCard(String cardNumber) {
        return cardNumber != null && cardNumber.length() >= 13;
    }
}