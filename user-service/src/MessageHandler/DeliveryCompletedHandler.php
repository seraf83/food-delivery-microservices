<?php

declare(strict_types=1);

namespace App\MessageHandler;

use App\Entity\Notification;
use App\Message\DeliveryCompleted;
use Doctrine\ORM\EntityManagerInterface;
use Symfony\Component\Messenger\Attribute\AsMessageHandler;

#[AsMessageHandler]
class DeliveryCompletedHandler
{
    public function __construct(private EntityManagerInterface $em) {}

    public function __invoke(DeliveryCompleted $event): void
    {
        $message = "🎉 Order #{$event->orderId} delivered! Enjoy your meal!";

        $notification = new Notification($event->userId, $message, 'delivery_completed');
        $this->em->persist($notification);
        $this->em->flush();

        echo "📱 SMS → user#{$event->userId}: {$message}\n";
    }
}
