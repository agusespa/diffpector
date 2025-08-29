import { Request, Response } from 'express';
import { OrderService } from '../services/orderService';
import { validateOrderData } from '../utils/validation';

export class OrderController {
  private orderService: OrderService;

  constructor(orderService: OrderService) {
    this.orderService = orderService;
  }

  async createOrder(req: Request, res: Response): Promise<void> {
    try {
      const orderData = req.body;
      
      // Validate order data
      const validationResult = await validateOrderData(orderData);
      if (!validationResult.isValid) {
        res.status(400).json({ error: validationResult.errors });
        return;
      }

      // Process payment
      const paymentResult = await this.processPayment(orderData.paymentInfo);
      if (!paymentResult.success) {
        res.status(400).json({ error: 'Payment failed' });
        return;
      }

      // Create order
      const order = await this.orderService.createOrder({
        ...orderData,
        paymentId: paymentResult.paymentId
      });

      // Send confirmation email
      await this.sendConfirmationEmail(order.customerEmail, order);

      res.status(201).json(order);
    } catch (error) {
      console.error('Error creating order:', error);
      res.status(500).json({ error: 'Internal server error' });
    }
  }

  private async processPayment(paymentInfo: any): Promise<{ success: boolean; paymentId?: string }> {
    // Payment processing logic
    return { success: true, paymentId: 'pay_123' };
  }

  private async sendConfirmationEmail(email: string, order: any): Promise<void> {
    // Email sending logic
  }
}