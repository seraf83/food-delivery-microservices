<?php

declare(strict_types=1);

namespace App\MessageHandler;

use App\Entity\Notification;
use App\Message\DeliveryAssigned;
use Doctrine\ORM\EntityManagerInterface;
use Symfony\Component\Messenger\Attribute\AsMessageHandler;

#[AsMessageHandler]
class DeliveryAssignedHandler
{
    public function __construct(private EntityManagerInterface $em) {}

    public function __invoke(DeliveryAssigned $event): void
    {
        $message = "🛵 Courier {$event->courierName} is on the way! Phone: {$event->courierPhone}. ETA: {$event->estimatedMinutes} min";

        $notification = new Notification(0, $message, 'delivery_assigned');
        $this->em->persist($notification);
        $this->em->flush();

        echo "📱 SMS → order#{$event->orderId}: {$message}\n";
    }
}
