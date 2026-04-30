<?php

declare(strict_types=1);

namespace App\MessageHandler;

use App\Entity\Order;
use App\Message\DeliveryCompleted;
use Doctrine\ORM\EntityManagerInterface;
use Symfony\Component\Messenger\Attribute\AsMessageHandler;

#[AsMessageHandler]
class DeliveryCompletedHandler
{
    public function __construct(private EntityManagerInterface $em) {}

    public function __invoke(DeliveryCompleted $event): void
    {
        $order = $this->em->find(Order::class, $event->orderId);
        if (!$order) return;

        $order->setStatus('delivered');
        $order->setDeliveredAt(new \DateTime($event->completedAt));
        $this->em->flush();

        echo "🎉 Order #{$event->orderId} — delivered!\n";
    }
}
