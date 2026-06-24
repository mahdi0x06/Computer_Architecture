    module RegisterFile (
    input clk,
    input regWrite,
    input [4:0] readReg1, readReg2, writeReg,
    input [31:0] writeData,
    output [31:0] readData1, readData2,
    output [31:0] R0, R1, R2, R3, R4, R5, R6, R7, R8, R9, R10, 
    output [31:0] R11, R12, R13, R14, R15, R16, R17, R18, R19, R20, 
    output [31:0] R21, R22, R23, R24, R25, R26, R27, R28, R29, R30, R31
);
    reg [31:0] registers [31:0];
    integer i;
    
    initial begin
        for (i = 0; i < 32; i = i + 1) begin
            registers[i] = 32'b0;
        end
    end
    
    assign readData1 = registers[readReg1];
    assign readData2 = registers[readReg2];
    
    assign {R31, R30, R29, R28, R27, R26, R25, R24, R23, R22, R21, R20, R19, R18, R17, R16, 
            R15, R14, R13, R12, R11, R10, R9, R8, R7, R6, R5, R4, R3, R2, R1, R0} = 
           {registers[31], registers[30], registers[29], registers[28], registers[27],
            registers[26], registers[25], registers[24], registers[23], registers[22],
            registers[21], registers[20], registers[19], registers[18], registers[17],
            registers[16], registers[15], registers[14], registers[13], registers[12],
            registers[11], registers[10], registers[9], registers[8], registers[7], 
            registers[6], registers[5], registers[4], registers[3], registers[2], 
            registers[1], registers[0]};

    always @(posedge clk) begin
        if (regWrite && (writeReg != 5'b0)) begin
            registers[writeReg] <= writeData;
        end
    end
endmodule

module Multiplier (
    input [31:0] a,
    input [31:0] b,
    output reg [31:0] result
);
    integer i;
    always @(*) begin
        result = 32'b0;
        for (i = 0; i < 32; i = i + 1) begin
            if (b[i]) result = result + (a << i);
        end
    end
endmodule

module Divider (
    input [31:0] a,
    input [31:0] b,
    output reg [31:0] result
);
    integer i;
    reg [31:0] quotient, remainder;
    always @(*) begin
        quotient = 32'b0;
        remainder = 32'b0;
        for (i = 31; i >= 0; i = i - 1) begin
            remainder = (remainder << 1) | ((a >> i) & 1'b1);
            if (remainder >= b && b != 0) begin
                remainder = remainder - b;
                quotient[i] = 1'b1;
            end
        end
        result = quotient;
    end
endmodule

module ALU (
    input [31:0] a,
    input [31:0] b,
    input [3:0] aluControl,
    output reg [31:0] result,
    output zero
);
    wire [31:0] mult_res, div_res;
    
    Multiplier m1 (a, b, mult_res);
    Divider d1 (a, b, div_res);
    
    always @(*) begin
        case (aluControl)
            4'b0000: result = a + b;
            4'b0001: result = a - b;
            4'b0100: result = mult_res;
            4'b0101: result = div_res;
            4'b0110: result = b << 16;
            default: result = 32'b0;
        endcase
    end
    
    assign zero = (result == 32'b0);
endmodule

module ControlUnit (
    input [5:0] opcode,
    input [5:0] funct,
    output reg regDst, regWrite, aluSrc, memRead, memWrite, branch, jump, jal, jr,
    output reg [1:0] memToReg,
    output reg [3:0] aluOp
);
    always @(*) begin
        regDst = 0; regWrite = 0; aluSrc = 0; memRead = 0; memWrite = 0; 
        memToReg = 0; branch = 0; jump = 0; jal = 0; jr = 0; aluOp = 4'b0000;
        
        case (opcode)
            6'b000000: begin
                if (funct == 6'b001000) begin
                    jr = 1;
                end else begin
                    regDst = 1; regWrite = 1;
                    case (funct)
                        6'b100000: aluOp = 4'b0000;
                        6'b100010: aluOp = 4'b0001;
                        6'b011000: aluOp = 4'b0100;
                        6'b011010: aluOp = 4'b0101;
                    endcase
                end
            end
            6'b100011: begin regWrite = 1; aluSrc = 1; memRead = 1; memToReg = 1; aluOp = 4'b0000; end
            6'b101011: begin aluSrc = 1; memWrite = 1; aluOp = 4'b0000; end
            6'b001000: begin regWrite = 1; aluSrc = 1; aluOp = 4'b0000; end
            6'b001001: begin regWrite = 1; aluSrc = 1; aluOp = 4'b0001; end
            6'b001111: begin regWrite = 1; aluSrc = 1; aluOp = 4'b0110; end
            6'b000100: begin branch = 1; aluOp = 4'b0001; end
            6'b000010: jump = 1;
            6'b000011: begin jump = 1; jal = 1; regWrite = 1; memToReg = 2; end
        endcase
    end
endmodule

module PC (
    input clk, 
    input rst, 
    input [31:0] nextPC,
    output reg [31:0] pc
);
    initial begin
        pc = 32'b0;
    end

    always @(posedge clk or posedge rst) begin
        if (rst) pc <= 32'b0; 
        else pc <= nextPC;
    end
endmodule

module AddressCalc (
    input [31:0] pc, 
    input [31:0] immExt, 
    input [25:0] jumpAddr, 
    input [31:0] rs_val,
    input jump, 
    input branch, 
    input zero, 
    input jr,
    output [31:0] nextPC
);
    wire [31:0] pcPlus4 = pc + 4;
    
    assign nextPC = jr ? rs_val :
                    jump ? {pcPlus4[31:28], jumpAddr, 2'b00} : 
                    (branch && zero) ? (pcPlus4 + (immExt << 2)) : pcPlus4;
endmodule

module main (
    input Clk, Rst,
    input [31:0] InstrIn, DataIn,
    output [31:0] InstrAddr, DataAddr,
    output DataWrite, 
    output [31:0] DataOut,
    output [31:0] R0, R1, R2, R3, R4, R5, R6, R7, R8, R9, R10, 
    output [31:0] R11, R12, R13, R14, R15, R16, R17, R18, R19, R20, 
    output [31:0] R21, R22, R23, R24, R25, R26, R27, R28, R29, R30, R31
);
    wire [31:0] pc, nextPC, instr, readData1, readData2, writeData, aluResult, immExt;
    wire [4:0] writeReg;
    wire regDst, regWrite, aluSrc, memRead, memWrite, branch, jump, jal, jr, zero;
    wire [1:0] memToReg;
    wire [3:0] aluOp;

    assign instr = InstrIn; 
    assign InstrAddr = pc >> 2; 
    assign DataAddr = aluResult >> 2; 
    assign DataWrite = memWrite; 
    assign DataOut = readData2;

    PC pc_unit (Clk, Rst, nextPC, pc);
    
    ControlUnit cu (
        .opcode(instr[31:26]), 
        .funct(instr[5:0]), 
        .regDst(regDst), 
        .regWrite(regWrite), 
        .aluSrc(aluSrc), 
        .memRead(memRead), 
        .memWrite(memWrite), 
        .branch(branch), 
        .jump(jump), 
        .jal(jal), 
        .jr(jr),
        .memToReg(memToReg), 
        .aluOp(aluOp)
    );

    assign writeReg = (jal) ? 5'd31 : (regDst ? instr[15:11] : instr[20:16]);
    
    RegisterFile rf (
        .clk(Clk), 
        .regWrite(regWrite), 
        .readReg1(instr[25:21]), 
        .readReg2(instr[20:16]), 
        .writeReg(writeReg), 
        .writeData(writeData), 
        .readData1(readData1), 
        .readData2(readData2),
        .R0(R0), .R1(R1), .R2(R2), .R3(R3), .R4(R4), .R5(R5), .R6(R6), .R7(R7), .R8(R8), .R9(R9), .R10(R10),
        .R11(R11), .R12(R12), .R13(R13), .R14(R14), .R15(R15), .R16(R16), .R17(R17), .R18(R18), .R19(R19), .R20(R20),
        .R21(R21), .R22(R22), .R23(R23), .R24(R24), .R25(R25), .R26(R26), .R27(R27), .R28(R28), .R29(R29), .R30(R30), .R31(R31)
    );

    assign immExt = {{16{instr[15]}}, instr[15:0]};
    ALU alu_unit (readData1, (aluSrc ? immExt : readData2), aluOp, aluResult, zero);
    
    AddressCalc addr_calc (pc, immExt, instr[25:0], readData1, jump, branch, zero, jr, nextPC);
    
    assign writeData = (memToReg == 2'b10) ? (pc + 4) : ((memToReg == 2'b01) ? DataIn : aluResult);
endmodule