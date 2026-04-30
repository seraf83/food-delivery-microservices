<?php

declare(strict_types=1);

namespace App\MessageHandler;

use App\Entity\Order;
use App\Message\DeliveryAssigned;
use Doctrine\ORM\EntityManagerInterface;
use Symfony\Component\Messenger\Attribute\AsMessageHandler;

#[AsMessageHandler]
class DeliveryAssignedHandler
{
    public function __construct(private EntityManagerInterface $em) {}

    public function __invoke(DeliveryAssigned $event): void
    {
        $order = $this->em->find(Order::class, $event->orderId);
        if (!$order) return;

        $order->setStatus('assigned');
        $order->setCourierId($event->courierId);
        $order->setEstimatedMinutes($event->estimatedMinutes);
        $this->em->flush();

        echo "✅ Order #{$event->orderId} — courier assigned: {$event->courierName}\n";
    }
}
