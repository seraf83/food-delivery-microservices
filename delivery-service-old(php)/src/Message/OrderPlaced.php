<?php

declare(strict_types=1);

namespace App\Message;

final class OrderPlaced
{
    public function __construct(
        public readonly int    $orderId,
        public readonly int    $userId,
        public readonly string $deliveryAddress,
        public readonly array  $items,
        public readonly float  $totalAmount,
    ) {}
}
