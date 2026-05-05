 import { MultiServerConfig } from "xypriss";
 import { xmsc } from "../configs/xms.config";
 export const mainServer: MultiServerConfig = {
     id: xmsc.main.id,
     port: xmsc.main.port,
     routePrefix: "/api",
 };