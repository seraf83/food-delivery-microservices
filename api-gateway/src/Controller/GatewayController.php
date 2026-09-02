<?php

declare(strict_types=1);

namespace App\Controller;

use App\Service\ProxyService;
use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;
use Symfony\Component\HttpFoundation\Request;
use Symfony\Component\HttpFoundation\Response;
use Symfony\Component\Routing\Attribute\Route;
use Lexik\Bundle\JWTAuthenticationBundle\Services\JWTTokenManagerInterface;
use Symfony\Component\Security\Core\Authentication\Token\Storage\TokenStorageInterface;


class GatewayController extends AbstractController
{
    public function __construct(
        private readonly ProxyService $proxyService,
        private readonly JWTTokenManagerInterface $jwtManager,
        private readonly TokenStorageInterface $tokenStorage
    )
    {

    }

    #[Route('/api/register', methods: ['POST'])]
    public function register(Request $request): Response
    {
        return $this->proxyService->proxy(
            url: $_ENV['USER_SERVICE_URL'] . '/api/register',
            request: $request,
        );
    }

    #[Route('/api/login', methods: ['POST'])]
    public function login(Request $request): Response
    {
        return $this->proxyService->proxy(
            url: $_ENV['USER_SERVICE_URL'] . '/api/login',
            request: $request,
        );
    }

    #[Route('/api/orders', methods: ['GET', 'POST'])]
    public function orders(Request $request): Response
    {
//        $user = $this->getUser();
//        dd($user->getUserIdentifier(), $user->getRoles(), $this->userHeaders());

        return $this->proxyService->proxy(
            url: $_ENV['ORDER_SERVICE_URL'] . '/api/orders',
            request: $request,
            extraHeaders: $this->userHeaders()
        );
    }

    #[Route('/api/me', methods: ['GET'])]
    public function me(Request $request): Response
    {
        $user = $this->getUser();

        return $this->proxyService->proxy(
            url: $_ENV['USER_SERVICE_URL'] . '/api/me',
            request: $request,
            extraHeaders: $this->userHeaders()
        );
    }

    #[Route('/api/deliveries/{orderId}/complete', methods: ['POST'])]
    public function completeDelivery(Request $request, int $orderId): Response
    {
        return $this->proxyService->proxy(
            url: $_ENV['DELIVERY_SERVICE_URL'] . '/deliveries/' . $orderId . '/complete',
            request: $request,
        );
    }

    private function userHeaders(): array
    {
        $decodedToken = $this->jwtManager->decode($this->tokenStorage->getToken());

        return [
            'X-User-Id'   => $decodedToken['id'],
            'X-User-Role' => $decodedToken['roles'][0],
        ];
    }
}
