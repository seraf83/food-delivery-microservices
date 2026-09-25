<?php

namespace Shared\Event;

final class OrderCreatedEvent
{
    public function __construct(
        public readonly string $orderId,
        public readonly string $userId,
        public readonly float $totalAmount,
        public readonly \DateTimeImmutable $createdAt,
    ) {
    }
}