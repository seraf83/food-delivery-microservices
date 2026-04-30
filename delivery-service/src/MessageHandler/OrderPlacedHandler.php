<?php

declare(strict_types=1);

namespace App\MessageHandler;

use App\Entity\Courier;
use App\Entity\Delivery;
use App\Message\DeliveryAssigned;
use App\Message\OrderPlaced;
use Doctrine\ORM\EntityManagerInterface;
use Symfony\Component\Messenger\Attribute\AsMessageHandler;
use Symfony\Component\Messenger\MessageBusInterface;

#[AsMessageHandler]
class OrderPlacedHandler
{
    public function __construct(
        private EntityManagerInterface $em,
        private MessageBusInterface    $bus,
    ) {}

    public function __invoke(OrderPlaced $event): void
    {
        echo "📦 New order received #{$event->orderId}\n";

        $courier = $this->em
            ->getRepository(Courier::class)
            ->findOneBy(['status' => 'available']);

        if (!$courier) {
            echo "⚠️  No available couriers for order #{$event->orderId}\n";
            return;
        }

        $delivery = new Delivery(
            orderId:   $event->orderId,
            userId:    $event->userId,
            courierId: $courier->getId(),
            address:   $event->deliveryAddress,
        );
        $this->em->persist($delivery);

        $courier->setStatus('busy');
        $this->em->flush();

        $this->bus->dispatch(new DeliveryAssigned(
            orderId:          $event->orderId,
            courierId:        $courier->getId(),
            courierName:      $courier->getName(),
            courierPhone:     $courier->getPhone(),
            estimatedMinutes: random_int(20, 45),
        ));

        echo "🛵 Courier {$courier->getName()} assigned to order #{$event->orderId}\n";
    }
}
