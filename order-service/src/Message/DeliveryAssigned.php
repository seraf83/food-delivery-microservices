<?php

declare(strict_types=1);

namespace App\Message;

final class DeliveryAssigned
{
    public function __construct(
        public readonly int    $orderId,
        public readonly int    $courierId,
        public readonly string $courierName,
        public readonly string $courierPhone,
        public readonly int    $estimatedMinutes,
        // Last and defaulted: the Kafka serializer maps fields by position,
        // and older messages were published without it.
        public readonly int    $userId = 0,
    ) {}
}
