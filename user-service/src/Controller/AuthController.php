<?php

declare(strict_types=1);

namespace App\Controller;

use App\Entity\User;
use Doctrine\ORM\EntityManagerInterface;
use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;
use Symfony\Component\HttpFoundation\JsonResponse;
use Symfony\Component\HttpFoundation\Request;
use Symfony\Component\PasswordHasher\Hasher\UserPasswordHasherInterface;
use Symfony\Component\Routing\Attribute\Route;

#[Route('/api')]
class AuthController extends AbstractController
{
    public function __construct(
        private readonly EntityManagerInterface      $em,
        private readonly UserPasswordHasherInterface $hasher,
    ) {}

    #[Route('/register', methods: ['POST'])]
    public function register(Request $request): JsonResponse
    {
        $data = json_decode($request->getContent(), true);

        if (empty($data['email']) || empty($data['password']) || empty($data['role'])) {
            return $this->json(['error' => 'email, password and role are required'], 400);
        }

        if (!in_array($data['role'], ['customer', 'courier'])) {
            return $this->json(['error' => 'role must be customer or courier'], 400);
        }

        $user = new User(
            email: $data['email'],
            password: '',
            role: $data['role'],
        );

        $user->setPassword(
            $this->hasher->hashPassword($user, $data['password'])
        );

        $this->em->persist($user);
        $this->em->flush();

        return $this->json(['message' => 'User registered successfully'], 201);
    }

    #[Route('/me', methods: ['GET'])]
    public function me(Request $request): JsonResponse
    {
        $userId = $request->headers->get('X-User-Id');
        $userRole = $request->headers->get('X-User-Role');

        return $this->json([
            'email' => $userId,
            'role'  => $userRole,
        ]);
    }
}
