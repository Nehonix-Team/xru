import { Router } from "xypriss";
export const mainRouter = Router();
mainRouter.get("/", (ctx) => { 
    ctx.body = { 
        message: "Welcome main", 
        mode: "xms" 
    }; 
});
