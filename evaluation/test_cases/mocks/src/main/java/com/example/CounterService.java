package com.example;

import java.util.concurrent.atomic.AtomicInteger;

public class CounterService {
    private AtomicInteger counter = new AtomicInteger(0);
    
    public void increment() {
        counter.incrementAndGet();
    }
    
    public void decrement() {
        counter.decrementAndGet();
    }
    
    public int getValue() {
        return counter.get();
    }
    
    public void reset() {
        counter.set(0);
    }
}