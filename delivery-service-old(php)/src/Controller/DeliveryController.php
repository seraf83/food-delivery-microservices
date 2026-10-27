<?php

declare(strict_types=1);

namespace App\Controller;

use App\Entity\Courier;
use App\Entity\Delivery;
use App\Message\DeliveryCompleted;
use Doctrine\ORM\EntityManagerInterface;
use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;
use Symfony\Component\HttpFoundation\JsonResponse;
use Symfony\Component\Messenger\MessageBusInterface;
use Symfony\Component\Routing\Attribute\Route;

class DeliveryController extends AbstractController
{
    public function __construct(
        private EntityManagerInterface $em,
        private MessageBusInterface    $bus,
    ) {}

    #[Route('/deliveries/{orderId}/complete', methods: ['POST'])]
    public function complete(int $orderId): JsonResponse
    {
        $delivery = $this->em->getRepository(Delivery::class)
            ->findOneBy(['orderId' => $orderId]);

        if (!$delivery) {
            return new JsonResponse(['error' => 'Not found'], 404);
        }

        $delivery->setStatus('completed');
        $delivery->setCompletedAt(new \DateTime());

        $courier = $this->em->find(Courier::class, $delivery->getCourierId());
        $courier?->setStatus('available');

        $this->em->flush();

        $this->bus->dispatch(new DeliveryCompleted(
            orderId:     $orderId,
            userId:      $delivery->getUserId(),
            completedAt: (new \DateTime())->format('Y-m-d H:i:s'),
        ));

        return new JsonResponse(['status' => 'completed']);
    }

    #[Route('/deliveries/{orderId}', methods: ['GET'])]
    public function status(int $orderId): JsonResponse
    {
        $delivery = $this->em->getRepository(Delivery::class)
            ->findOneBy(['orderId' => $orderId]);

        if (!$delivery) {
            return new JsonResponse(['error' => 'Not found'], 404);
        }

        return new JsonResponse($delivery->toArray());
    }

    #[Route('/couriers', methods: ['GET'])]
    public function couriers(): JsonResponse
    {
        $couriers = $this->em->getRepository(Courier::class)->findAll();
        return new JsonResponse(array_map(
            fn($c) => [
                'id'     => $c->getId(),
                'name'   => $c->getName(),
                'status' => $c->getStatus(),
            ],
            $couriers
        ));
    }
}
