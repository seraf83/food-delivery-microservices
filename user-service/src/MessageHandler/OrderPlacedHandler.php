<?php

declare(strict_types=1);

namespace App\MessageHandler;

use App\Entity\Notification;
use App\Message\OrderPlaced;
use Doctrine\ORM\EntityManagerInterface;
use Symfony\Component\Messenger\Attribute\AsMessageHandler;

#[AsMessageHandler]
class OrderPlacedHandler
{
    public function __construct(private EntityManagerInterface $em) {}

    public function __invoke(OrderPlaced $event): void
    {
        $items = implode(', ', array_map(
            fn($i) => "{$i['name']} x{$i['qty']}",
            $event->items
        ));

        $message = "✅ Order #{$event->orderId} accepted! Items: {$items}. Total: {$event->totalAmount} UAH";

        $notification = new Notification($event->userId, $message, 'order_placed');
        $this->em->persist($notification);
        $this->em->flush();

        echo "📱 SMS → user#{$event->userId}: {$message}\n";
    }
}
