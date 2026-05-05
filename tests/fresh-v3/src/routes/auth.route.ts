import { Router } from "xypriss";
export const authRouter = Router();
authRouter.get("/", (ctx) => { 
    ctx.body = { 
        message: "Welcome auth", 
        mode: "xms" 
    }; 
});
