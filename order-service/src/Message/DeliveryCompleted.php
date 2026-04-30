<?php

declare(strict_types=1);

namespace App\Message;

final class DeliveryCompleted
{
    public function __construct(
        public readonly int    $orderId,
        public readonly int    $userId,
        public readonly string $completedAt,
    ) {}
}
