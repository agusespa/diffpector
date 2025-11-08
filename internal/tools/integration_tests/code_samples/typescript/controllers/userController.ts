import { Request, Response } from 'express';
import { UserService } from '../services/userService';

/**
 * UserController handles HTTP requests related to users.
 */
export class UserController {
    private userService: UserService;

    constructor(userService: UserService) {
        this.userService = userService;
    }

    /**
     * Handles GET /api/users/:id requests.
     */
    public async getUser(req: Request, res: Response): Promise<void> {
        try {
            const userId = req.params.id;
            if (!userId) {
                res.status(400).json({ error: 'User ID is required' });
                return;
            }

            const user = await this.userService.getUser(userId);
            res.status(200).json(user);
        } catch (error) {
            console.error('Handler failed to fulfill request:', error);
            res.status(500).json({ error: 'Internal server error' });
        }
    }

    public async getUserCount(req: Request, res: Response): Promise<void> {
        try {
            const count = await this.userService.getTotalUserCount();
            res.status(200).json({ count });
        } catch (error) {
            res.status(500).json({ error: 'Internal server error' });
        }
    }
}
