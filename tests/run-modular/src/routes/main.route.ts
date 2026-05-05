import { Router } from "xypriss";

/**
 * Default Route Module.
 * Handles the main application logic for the standalone server.
 */
export const mainRouter = Router();

mainRouter.get("/", (ctx) => { 
    ctx.body = { 
        status: "online",
        message: "Standalone server is running",
        timestamp: new Date().toISOString()
    }; 
});
