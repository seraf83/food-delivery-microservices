<?php

declare(strict_types=1);

namespace App\Controller;

use App\Entity\Order;
use App\Message\OrderPlaced;
use Doctrine\ORM\EntityManagerInterface;
use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;
use Symfony\Component\HttpFoundation\JsonResponse;
use Symfony\Component\HttpFoundation\Request;
use Symfony\Component\Messenger\MessageBusInterface;
use Symfony\Component\Routing\Attribute\Route;

class OrderController extends AbstractController
{
    public function __construct(
        private EntityManagerInterface $em,
        private MessageBusInterface    $bus,
    ) {}

    #[Route('/orders', methods: ['POST'])]
    public function create(Request $request): JsonResponse
    {
        $data = json_decode($request->getContent(), true);

        $order = new Order(
            userId:          $data['user_id'],
            deliveryAddress: $data['address'],
            items:           $data['items'],
            totalAmount:     $data['total'],
        );

        $this->em->persist($order);
        $this->em->flush();

        $this->bus->dispatch(new OrderPlaced(
            orderId:         $order->getId(),
            userId:          $order->getUserId(),
            deliveryAddress: $order->getDeliveryAddress(),
            items:           $order->getItems(),
            totalAmount:     $order->getTotalAmount(),
        ));

        return new JsonResponse($order->toArray(), 201);
    }

    #[Route('/orders/{id}', methods: ['GET'])]
    public function show(int $id): JsonResponse
    {
        $order = $this->em->find(Order::class, $id);

        if (!$order) {
            return new JsonResponse(['error' => 'Not found'], 404);
        }

        return new JsonResponse($order->toArray());
    }
}
