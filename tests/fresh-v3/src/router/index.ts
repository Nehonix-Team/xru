import { Router } from "xypriss";
 import { authRouter } from "../routes/auth.route";
 import { mainRouter } from "../routes/main.route";

/**
 * Main Application Router
 */
const router = Router();

 router.use("/auth", authRouter);
 router.use("/api", mainRouter);

export default router;
