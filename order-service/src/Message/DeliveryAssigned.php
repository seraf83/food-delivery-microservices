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
    ) {}
}
