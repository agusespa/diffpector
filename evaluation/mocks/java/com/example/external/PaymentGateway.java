package com.example.external;

public interface PaymentGateway {
    /**
     * Charges the user's card.
     * @param token Card token
     * @param amount Amount in cents
     * @return Transaction ID
     * @throws PaymentDeclinedException if charge fails
     */
    String chargeCard(String token, long amount);
}
